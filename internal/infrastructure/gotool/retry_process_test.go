// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

func TestRetryProcessHelper(t *testing.T) {
	if os.Getenv("COVERLINT_PROCESS_HELPER") != "1" {
		t.Parallel()
		return
	}
	os.Exit(7)
}

func failingCommand(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRetryProcessHelper$")
	cmd.Env = append(os.Environ(), "COVERLINT_PROCESS_HELPER=1")
	return cmd
}

func TestStartUsesFreshCommandsAndWaitsOnce(t *testing.T) {
	t.Parallel()
	var commands []*exec.Cmd
	starts := 0
	cmd, err := startCommand(t.Context(), func() *exec.Cmd {
		cmd := failingCommand(t.Context())
		commands = append(commands, cmd)
		return cmd
	}, func(cmd *exec.Cmd) error {
		starts++
		if starts < 3 {
			return &os.PathError{Op: "fork", Path: cmd.Path, Err: syscall.EAGAIN}
		}
		return cmd.Start()
	})
	if err != nil {
		t.Fatal(err)
	}
	if starts != 3 || commands[0] == commands[1] || commands[1] == commands[2] {
		t.Fatalf("commands were reused: %v, starts %d", commands, starts)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("expected process exit error")
	}
}

func TestRunCommandDoesNotRetryExitFailure(t *testing.T) {
	t.Parallel()
	calls := 0
	err := runCommand(t.Context(), func() *exec.Cmd { calls++; return failingCommand(t.Context()) })
	var exitErr *exec.ExitError
	if calls != 1 || !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("run error = %v, calls %d", err, calls)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = runCommand(ctx, func() *exec.Cmd { calls++; return failingCommand(ctx) })
	if calls != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v, calls %d", err, calls)
	}
}

func TestToolOperationsNeverRetryStartedProcesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "calls")
	script := `#!/bin/sh
printf '%s\n' "$1" >> "$COVERLINT_CALLS"
case "$1" in
test)
  printf 'mode: atomic\nexample.com/fixture/calc.go:1.1,1.2 1 0\n' > "$5"
  if [ "$COVERLINT_BAD_PROFILE" = 1 ]; then printf 'bad profile\n' > "$5"; fi
  printf '%s\n' '{"Action":"fail","Package":"example.com/fixture","Test":"TestFail"}'
  exit 1
  ;;
list)
  printf '{bad json'
  exit 0
  ;;
tool)
  exit 1
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("COVERLINT_CALLS", marker)
	tool := New()
	coverage, err := tool.Collect(
		t.Context(),
		&outbound.CoverageRequest{Patterns: []string{"./..."}},
	)
	if err != nil || !coverage.TestsFailed || len(coverage.Failures) == 0 {
		t.Fatalf("failed tests should remain a soft failure: %#v, %v", coverage, err)
	}
	t.Setenv("COVERLINT_BAD_PROFILE", "1")
	if _, err := tool.Collect(
		t.Context(),
		&outbound.CoverageRequest{Patterns: []string{"./..."}},
	); err == nil {
		t.Fatal("expected invalid coverage profile to remain a hard failure")
	}
	if _, err := tool.List(
		t.Context(),
		&outbound.PackageRequest{Patterns: []string{"./..."}},
	); err == nil {
		t.Fatal("expected JSON decode error")
	}
	err = tool.Open(
		t.Context(),
		&outbound.HTMLOpenRequest{
			Profile: coverage.Profile,
			Stdout:  io.Discard,
			Stderr:  io.Discard,
		},
	)
	if err == nil {
		t.Fatal("expected HTML command failure")
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "test\ntest\nlist\ntool\n" {
		t.Fatalf("executions = %q, %v", data, err)
	}
}

func TestStartPermanentFailure(t *testing.T) {
	t.Parallel()
	calls := 0
	cmd, err := startCommand(t.Context(), func() *exec.Cmd { return failingCommand(t.Context()) },
		func(*exec.Cmd) error { calls++; return os.ErrPermission })
	if cmd != nil || calls != 1 || !errors.Is(err, os.ErrPermission) ||
		!strings.Contains(err.Error(), "start process") {
		t.Fatalf("start = %v, %v, calls %d", cmd, err, calls)
	}
}

func TestListMissingExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	packages, err := New().List(t.Context(), &outbound.PackageRequest{Patterns: []string{"."}})
	if packages != nil || !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("missing executable = %v, %v", packages, err)
	}
}

func TestReadProfileCloseFailureIsPermanent(t *testing.T) {
	t.Parallel()
	file := createClosedTempFile(t)
	calls := 0
	err := retry.Run(
		t.Context(),
		func() error { calls++; _, err := readAndCloseProfile(file); return err },
	)
	if calls != 1 || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("close failure = %v, calls %d", err, calls)
	}
	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("coverage caller stopped")
	cancel(cause)
	if err := goCoverError(ctx, os.ErrPermission); !errors.Is(err, cause) {
		t.Fatalf("cover cause = %v", err)
	}
}

func TestTemporaryCleanupAfterCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	temp := t.TempDir()
	marker := filepath.Join(dir, "started")
	script := "#!/bin/sh\nprintf started > \"$COVERLINT_STARTED\"\nwhile :; do :; done\n"
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TMPDIR", temp)
	t.Setenv("COVERLINT_STARTED", marker)
	for _, open := range []bool{false, true} {
		ctx, stop := context.WithTimeout(t.Context(), 30*time.Second)
		t.Cleanup(stop)
		ctx, cancel := context.WithCancelCause(ctx)
		cause := errors.New("cancel after process start")
		done := make(chan error, 1)
		go func() {
			if open {
				done <- New().Open(ctx, &outbound.HTMLOpenRequest{Profile: []byte("mode: atomic\n"), Stdout: io.Discard, Stderr: io.Discard})
			} else {
				_, err := New().Collect(ctx, &outbound.CoverageRequest{Patterns: []string{"."}})
				done <- err
			}
		}()
		waitForProcessMarker(t, ctx, marker)
		cancel(cause)
		err := <-done
		stop()
		if !errors.Is(err, cause) {
			t.Fatalf("operation canceled = %v", err)
		}
		entries, err := os.ReadDir(temp)
		if err != nil || len(entries) != 0 {
			t.Fatalf("temporary files leaked: %v, %v", entries, err)
		}
		if err := os.Remove(marker); err != nil {
			t.Fatal(err)
		}
	}
}

func waitForProcessMarker(t *testing.T, ctx context.Context, marker string) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("process never signaled startup: %v", context.Cause(ctx))
		case <-ticker.C:
			_, err := os.Stat(marker)
			if err == nil {
				return
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
	}
}
