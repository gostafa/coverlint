// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v7"
	"github.com/gostafa/singleton"
)

func TestRetryRecoveryAndExhaustion(t *testing.T) {
	t.Parallel()
	for _, code := range []syscall.Errno{syscall.EINTR, syscall.EAGAIN, syscall.EMFILE, syscall.ENFILE, syscall.ETIMEDOUT} {
		t.Run(code.Error(), func(t *testing.T) {
			t.Parallel()
			calls := 0
			var value string
			err := Run(t.Context(), func() error {
				calls++
				if calls < 3 {
					return &os.PathError{Op: "open", Path: "report", Err: code}
				}
				value = "complete"
				return nil
			})
			if err != nil || value != "complete" || calls != 3 {
				t.Fatalf("recovery = %q, %v, calls %d", value, err, calls)
			}
			calls = 0
			err = Run(t.Context(), func() error { calls++; return code })
			if calls != 3 || !errors.Is(err, code) {
				t.Fatalf("exhaustion = %q, %v, calls %d", value, err, calls)
			}
			assertNoRetryError(t, err)
		})
	}
}

func TestPermanentAndStoppedFailures(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{
		os.ErrNotExist, os.ErrPermission, syscall.EINVAL, io.ErrUnexpectedEOF,
		context.Canceled, context.DeadlineExceeded, Stop(syscall.EAGAIN), Stop(os.ErrNotExist),
		errors.Join(context.Canceled, syscall.EAGAIN),
	} {
		calls := 0
		expected := failure
		if stopped, ok := errors.AsType[*singleton.PermanentError](failure); ok {
			expected = stopped.Err
		}
		err := Run(t.Context(), func() error { calls++; return failure })
		if calls != 1 || !errors.Is(err, expected) {
			t.Fatalf("permanent = %v, calls %d (cause %v)", err, calls, failure)
		}
		assertNoRetryError(t, err)
	}
	if Stop(nil) != nil {
		t.Fatal("Stop(nil) != nil")
	}
}

func TestCancellationBeforeAndDuringWait(t *testing.T) {
	t.Parallel()
	cause := errors.New("caller stopped")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)
	calls := 0
	err := Run(ctx, func() error { calls++; return nil })
	if calls != 0 || !errors.Is(err, cause) {
		t.Fatalf("pre-cancellation = %v, calls %d", err, calls)
	}
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer stop()
	err = Run(ctx, func() error { calls++; return syscall.EAGAIN })
	if calls != 1 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait cancellation = %v, calls %d", err, calls)
	}
	assertNoRetryError(t, err)
}

func TestElapsedBudgetStopsScheduling(t *testing.T) {
	t.Parallel()
	calls := 0
	err := Run(t.Context(), func() error {
		calls++
		time.Sleep(1100 * time.Millisecond)
		return syscall.EAGAIN
	})
	if calls != 2 || !errors.Is(err, syscall.EAGAIN) {
		t.Fatalf("budget = %v, calls %d", err, calls)
	}
}

type testReader struct {
	io.Reader
	closeErr error
	closes   *int
}

func (reader *testReader) Close() error { *reader.closes++; return reader.closeErr }

type partialReader struct{ read bool }

func (reader *partialReader) Read(data []byte) (int, error) {
	if !reader.read {
		reader.read = true
		return copy(data, "partial"), syscall.EAGAIN
	}
	return 0, io.EOF
}

func TestReadRestartsAndDiscardsPartialData(t *testing.T) {
	t.Parallel()
	opens, closes := 0, 0
	var data []byte
	err := readWith(t.Context(), func() (io.ReadCloser, error) {
		opens++
		if opens == 1 {
			return nil, syscall.EMFILE
		}
		if opens == 2 {
			return &testReader{Reader: &partialReader{}, closes: &closes}, nil
		}
		return &testReader{Reader: strings.NewReader("complete"), closes: &closes}, nil
	}, func(reader io.Reader) error { var err error; data, err = io.ReadAll(reader); return err })
	if err != nil || string(data) != "complete" || opens != 3 || closes != 2 {
		t.Fatalf("read = %q, %v, opens %d closes %d", data, err, opens, closes)
	}
}

func TestCloseFailureStopsReadRetries(t *testing.T) {
	t.Parallel()
	for _, reader := range []io.Reader{strings.NewReader("complete"), &partialReader{}} {
		opens, closes := 0, 0
		err := readWith(t.Context(), func() (io.ReadCloser, error) {
			opens++
			return &testReader{Reader: reader, closes: &closes, closeErr: syscall.EAGAIN}, nil
		}, func(reader io.Reader) error { _, err := io.ReadAll(reader); return err })
		if opens != 1 || closes != 1 || !errors.Is(err, syscall.EAGAIN) {
			t.Fatalf("close failure = %v, opens %d closes %d", err, opens, closes)
		}
		assertNoRetryError(t, err)
	}
}

func TestFileAndTempOperations(t *testing.T) {
	t.Parallel()
	file, err := CreateTemp(t.Context(), "coverlint-retry-*")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(path); err != nil {
			t.Error(err)
		}
	})
	if err := os.WriteFile(path, []byte("report"), 0o600); err != nil {
		t.Fatal(err)
	}
	var data []byte
	err = ReadFile(
		t.Context(),
		path,
		func(reader io.Reader) error { var err error; data, err = io.ReadAll(reader); return err },
	)
	if err != nil || string(data) != "report" {
		t.Fatalf("read = %q, %v", data, err)
	}
	err = ReadFile(
		t.Context(),
		filepath.Join(t.TempDir(), "missing"),
		func(io.Reader) error { t.Fatal("read missing file"); return nil },
	)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = CreateTemp(ctx, "coverlint-retry-*")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("create canceled = %v", err)
	}
	if err := Run(t.Context(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func assertNoRetryError(t *testing.T, err error) {
	t.Helper()
	var failure *backoff.RetryError
	var stopped *singleton.PermanentError
	if errors.As(err, &failure) || errors.As(err, &stopped) {
		t.Fatalf("internal error escaped: %v", err)
	}
}

func TestCreateTempFailure(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMPDIR", missing)
	t.Setenv("TMP", missing)
	t.Setenv("TEMP", missing)
	file, err := CreateTemp(t.Context(), "coverlint-retry-*")
	if file != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("create = %v, %v", file, err)
	}
	assertNoRetryError(t, err)
}
