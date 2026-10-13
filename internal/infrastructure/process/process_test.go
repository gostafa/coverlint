// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestStartupRetriesFreshCommandsAndPermanentFailures(t *testing.T) {
	t.Parallel()
	commands := []*exec.Cmd{}
	calls := 0
	command, err := StartCommand(t.Context(), func() *exec.Cmd {
		cmd := exec.CommandContext(t.Context(), "go", "version")
		commands = append(commands, cmd)
		return cmd
	}, func(*exec.Cmd) error {
		calls++
		if calls == 1 {
			return syscall.EAGAIN
		}
		return nil
	})
	if err != nil || calls != 2 || command != commands[1] || commands[0] == commands[1] {
		t.Fatalf("start=%v %v calls=%d", command, err, calls)
	}
	_, err = StartCommand(
		t.Context(),
		func() *exec.Cmd { return exec.CommandContext(t.Context(), "go", "version") },
		func(*exec.Cmd) error { return os.ErrPermission },
	)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatal(err)
	}
}

func TestRunWaitsOnceAndReportsStartupAndExecutionFailures(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		program string
		args    []string
		failed  bool
	}{{"go", []string{"version"}, false}, {"go", []string{"nonexistent-subcommand"}, true}, {filepath.Join(t.TempDir(), "missing"), nil, true}} {
		err := RunCommand(
			t.Context(),
			func() *exec.Cmd { return exec.CommandContext(t.Context(), test.program, test.args...) },
		)
		if (err != nil) != test.failed {
			t.Fatalf("run=%v expected failure=%v", err, test.failed)
		}
	}
}

func TestCleanupSurvivesCancellationAndCloseErrors(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "temporary")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = CloseFile(file); err != nil {
		t.Fatal(err)
	}
	if err = CloseFile(file); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("close=%v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err = RemoveFile(ctx, path); err != nil {
		t.Fatal(err)
	}
	if err = RemoveFile(ctx, path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remove=%v", err)
	}
}
