// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

type resultWriter struct {
	written            int
	writeErr, closeErr error
	writes, closes     int
}

func (writer *resultWriter) Write([]byte) (int, error) {
	writer.writes++
	return writer.written, writer.writeErr
}
func (writer *resultWriter) Close() error { writer.closes++; return writer.closeErr }

func TestResultCloseFailureStopsRetries(t *testing.T) {
	t.Parallel()
	writer := &resultWriter{written: 1, writeErr: syscall.EINTR, closeErr: syscall.EAGAIN}
	err := retry.Run(
		t.Context(),
		func() error { return writeAndCloseResult(writer, []byte("data")) },
	)
	if writer.writes != 1 || writer.closes != 1 || !errors.Is(err, syscall.EINTR) ||
		!errors.Is(err, syscall.EAGAIN) {
		t.Fatalf("write/close = %#v, %v", writer, err)
	}
	writer = &resultWriter{written: 1}
	err = writeAndCloseResult(writer, []byte("data"))
	if !errors.Is(err, io.ErrShortWrite) || writer.closes != 1 {
		t.Fatalf("short write = %v", err)
	}
	writer = &resultWriter{writeErr: os.ErrPermission}
	if err := writeAndCloseResult(writer, []byte("data")); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("write = %v", err)
	}
}

func TestCompleteWritesReplacePartialContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "report")
	calls := 0
	err := retry.Run(t.Context(), func() error {
		calls++
		if calls == 1 {
			if err := os.WriteFile(
				path,
				[]byte("partial stale trailing bytes"),
				0o600,
			); err != nil {
				return err
			}
			return syscall.EAGAIN
		}
		return writeResultFile(path, []byte("complete"))
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "complete" || calls != 2 {
		t.Fatalf("data = %q, %v, calls %d", data, err, calls)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = writeOptionalResultFileContext(ctx, path, []byte("changed"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled write = %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "complete" {
		t.Fatalf("canceled write changed data: %q, %v", data, err)
	}
}
