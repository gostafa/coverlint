// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reportingports "github.com/gostafa/coverlint/internal/features/reporting/ports"
	"github.com/gostafa/coverlint/internal/infrastructure/process"
)

func TestHTMLPortExecutionAndCleanup(t *testing.T) {
	if err := New().Open(t.Context(), &reportingports.HTMLOpenRequest{}); !errors.Is(
		err,
		reportingports.ErrReporting,
	) {
		t.Fatalf("empty profile=%v", err)
	}
	t.Setenv("BROWSER", "true")
	dir := t.TempDir()
	file := filepath.Join(dir, "calc.go")
	if err := os.WriteFile(
		file,
		[]byte("package calc\nfunc Add(a,b int)int{return a+b}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	profile := []byte("mode: atomic\n" + filepath.ToSlash(file) + ":2.22,2.34 1 1\n")
	err := New().Open(t.Context(), &reportingports.HTMLOpenRequest{Profile: profile, Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		t.Fatalf("HTML execution: %v", err)
	}
	err = New().Open(t.Context(), &reportingports.HTMLOpenRequest{Profile: []byte("mode: atomic\nmissing.go:1.1,1.2 1 1\n"), Stdout: io.Discard, Stderr: io.Discard})
	if err == nil {
		t.Fatal("missing source was accepted")
	}
	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("caller stopped")
	cancel(cause)
	if err := goCoverError(ctx, os.ErrPermission); !errors.Is(err, cause) {
		t.Fatalf("cancel cause=%v", err)
	}
	t.Setenv("PATH", t.TempDir())
	err = New().Open(t.Context(), &reportingports.HTMLOpenRequest{Profile: profile, Stdout: io.Discard, Stderr: io.Discard})
	if err == nil {
		t.Fatal("missing executable was accepted")
	}
}

var (
	errCloseBoom = errors.New("close boom")
	errWriteBoom = errors.New("write boom")
)

func TestCloseHTMLInputErrBranches(t *testing.T) {
	t.Parallel()

	if err := closeHTMLInputErr(createTempFile(t)); err != nil {
		t.Fatalf("closeHTMLInputErr() success path: %v", err)
	}

	if err := closeHTMLInputErr(createClosedTempFile(t)); err == nil {
		t.Fatal("closeHTMLInputErr() error = nil, want close failure")
	}
}

func TestCloseHTMLInputHelpers(t *testing.T) {
	t.Parallel()

	file := createClosedTempFile(t)

	if err := closeHTMLInput(file); err == nil {
		t.Fatal("closeHTMLInput() error = nil, want close failure")
	}

	openFile := createTempFile(t)
	if err := closeHTMLInput(openFile); err != nil {
		t.Fatalf("closeHTMLInput() open file: %v", err)
	}
}

func TestCreateHTMLProfileFileTMPDIRFailure(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(blocked, "missing"))

	_, _, err := createHTMLProfileFileContext(t.Context())
	if err == nil || !strings.Contains(err.Error(), "create temporary HTML coverage input") {
		t.Fatalf("error = %v, want HTML create wrapper", err)
	}
}

func TestFinishHTMLReportWriteFailure(t *testing.T) {
	t.Parallel()

	err := finishHTMLReport(
		t.Context(),
		createClosedTempFile(t),
		&reportingports.HTMLOpenRequest{
			Profile: []byte("mode: atomic\n"),
			Stdout:  io.Discard,
			Stderr:  io.Discard,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want finish HTML wrapper", err)
	}
}

func TestJoinWriteCloseError(t *testing.T) {
	t.Parallel()

	err := joinWriteCloseError(createTempFile(t), 3, errWriteBoom)
	if err == nil || !errors.Is(err, errWriteBoom) {
		t.Fatalf("error = %v, want write boom", err)
	}

	if !strings.Contains(err.Error(), "wrote 3") {
		t.Fatalf("error = %v, want written count", err)
	}
}

func TestOpenHTMLReportCreateFailure(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(blocked, "missing"))

	err := openHTMLReport(t.Context(), &reportingports.HTMLOpenRequest{
		Profile: []byte("mode: atomic\n"),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want open HTML wrapper", err)
	}
}

func TestWriteAndCloseProfileCloseFailure(t *testing.T) {
	file := createTempFile(t)
	err := writeAndCloseProfileWith(file, []byte("mode: atomic\n"), func(closed *os.File) error {
		_ = process.CloseFile(closed)

		return errCloseBoom
	})
	if err == nil || !errors.Is(err, errCloseBoom) {
		t.Fatalf("error = %v, want close boom", err)
	}
}

func TestWriteAndCloseProfileWriteFailure(t *testing.T) {
	t.Parallel()

	err := writeAndCloseProfile(createClosedTempFile(t), []byte("mode: atomic\n"))
	if err == nil || !strings.Contains(err.Error(), "write temporary HTML coverage input") {
		t.Fatalf("error = %v, want write wrapper", err)
	}
}

func createClosedTempFile(t *testing.T) *os.File {
	t.Helper()

	file := createTempFile(t)
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	return file
}

func createTempFile(t *testing.T) *os.File {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "gotool-*.tmp")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}

	t.Cleanup(func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	})

	return file
}
