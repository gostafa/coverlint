// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

func prepareGoList(ctx context.Context, request *outbound.PackageRequest) (*goListRun, error) {
	cmd := goListCommand(ctx, request)
	stderr := NewCappedBuffer(commandOutputLimit)

	cmd.Stderr = &stderr

	run, err := startGoListCommand(cmd, &stderr)
	if err != nil {
		return nil, fmt.Errorf("prepare package list: %w", err)
	}

	return run, nil
}

func runCommand(ctx context.Context, command func() *exec.Cmd) error {
	cmd, err := startCommand(ctx, command, (*exec.Cmd).Start)
	if err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("wait command: %w", err)
	}

	return nil
}

func startCommand(
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

func startPackageList(ctx context.Context, request *outbound.PackageRequest) (*goListRun, error) {
	var run *goListRun

	err := retry.Run(ctx, func() error {
		var err error

		run, err = prepareGoList(ctx, request)
		if err != nil {
			return fmt.Errorf("prepare list attempt: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("start package list: %w", err)
	}

	return run, nil
}

func startGoListCommand(cmd *exec.Cmd, stderr *CappedBuffer) (*goListRun, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare go list: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start go list: %w", err)
	}

	return &goListRun{cmd: cmd, stdout: stdout, stderr: stderr}, nil
}
