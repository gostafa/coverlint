// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

// CloseFile closes a handle once; close failures are never retried.
func CloseFile(file *os.File) error {
	err := file.Close()
	if err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

// RemoveFile retries safe cleanup independently of caller cancellation.
func RemoveFile(ctx context.Context, name string) error {
	err := retry.Run(context.WithoutCancel(ctx), func() error {
		err := os.Remove(name)
		if err != nil {
			return fmt.Errorf("remove path: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("remove temporary file: %w", err)
	}

	return nil
}

// RunCommand retries startup, then waits once without rerunning the command.
func RunCommand(ctx context.Context, command func() *exec.Cmd) error {
	cmd, err := StartCommand(ctx, command, (*exec.Cmd).Start)
	if err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("wait command: %w", err)
	}

	return nil
}

// StartCommand recreates a command on each safe startup retry.
func StartCommand(
	ctx context.Context, command func() *exec.Cmd, start func(*exec.Cmd) error,
) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	err := retry.Run(ctx, func() error {
		cmd = command()

		err := start(cmd)
		if err != nil {
			return fmt.Errorf("start attempt: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	return cmd, nil
}
