// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gostafa/coverlint/internal/features/policy/ports"
	"github.com/gostafa/coverlint/internal/infrastructure/failure"
	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

func completeResultWrite(written, length int, err error) error {
	if err != nil {
		return fmt.Errorf(errWriteResultDataFormat, err)
	}

	if written != length {
		return fmt.Errorf(errWriteResultDataFormat, io.ErrShortWrite)
	}

	return nil
}

func ensureResultParentDirContext(ctx context.Context, path string) error {
	parent := filepath.Dir(path)

	if parent == emptyString || parent == currentDir {
		return nil
	}

	err := retry.Run(ctx, func() error {
		err := os.MkdirAll(parent, resultDirPerm)
		if err != nil {
			return fmt.Errorf("create result directory: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("create result directory %q: %w", parent, err)
	}

	return nil
}

func writeAndCloseResult(file io.WriteCloser, data []byte) error {
	written, writeErr := file.Write(data)

	closeErr := file.Close()
	if closeErr != nil {
		return fmt.Errorf(
			"finish result write: %w",
			retry.Stop(errors.Join(writeErr, fmt.Errorf("close result file: %w", closeErr))),
		)
	}

	err := completeResultWrite(written, len(data), writeErr)
	if err != nil {
		return fmt.Errorf(errWriteResultDataFormat, err)
	}

	return nil
}

func writeOptionalResultFileContext(ctx context.Context, path string, data []byte) error {
	if path == emptyString {
		return nil
	}

	err := ensureResultParentDirContext(ctx, path)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	err = retry.Run(ctx, func() error { return writeResultFile(path, data) })
	if err != nil {
		return fmt.Errorf("write result file %q: %w", path, err)
	}

	return nil
}

func writeResultFile(path string, data []byte) error {
	file, err := os.OpenFile(
		filepath.Clean(path),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		resultFilePerm,
	)
	if err != nil {
		return fmt.Errorf("open result file: %w", err)
	}

	return errors.Join(writeAndCloseResult(file, data))
}

// Write replaces one complete artifact and publishes stable persistence failures.
func (*Adapter) Write(ctx context.Context, path string, data []byte) error {
	return errors.Join(
		failure.Translate(writeOptionalResultFileContext(ctx, path, data), ports.ErrResultWrite),
	)
}
