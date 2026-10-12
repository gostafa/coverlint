// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateTemp retries creation only; closing the returned file is the caller's responsibility.
func CreateTemp(ctx context.Context, pattern string) (*os.File, error) {
	var file *os.File

	err := Run(ctx, func() error {
		var err error

		file, err = os.CreateTemp("", pattern)

		return wrapCreateError(err)
	})
	if err != nil {
		return nil, fmt.Errorf("create temporary file: %w", err)
	}

	return file, nil
}

// ReadFile retries open/read failures by reopening at offset zero. The callback
// must create fresh parser state each time and discard its results on failure.
// Close errors always stop retries, even when the read also failed transiently.
func ReadFile(ctx context.Context, path string, read func(io.Reader) error) error {
	err := readWith(ctx, func() (io.ReadCloser, error) {
		file, err := os.Open(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("open file %q: %w", path, err)
		}

		return file, nil
	}, read)
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}

	return nil
}

func finishRead(readErr, closeErr error) error {
	if closeErr != nil {
		return fmt.Errorf(
			"finish read: %w",
			Stop(errors.Join(readErr, fmt.Errorf("close file: %w", closeErr))),
		)
	}

	return readErr
}

func readAttempt(open func() (io.ReadCloser, error), read func(io.Reader) error) error {
	file, err := open()
	if err != nil {
		return fmt.Errorf("open reader: %w", err)
	}

	readErr := read(file)
	closeErr := file.Close()

	err = finishRead(readErr, closeErr)
	if err != nil {
		return fmt.Errorf("read attempt: %w", err)
	}

	return nil
}

func readWith(
	ctx context.Context,
	open func() (io.ReadCloser, error),
	read func(io.Reader) error,
) error {
	err := Run(ctx, func() error { return readAttempt(open, read) })
	if err != nil {
		return fmt.Errorf("read reader: %w", err)
	}

	return nil
}

func wrapCreateError(err error) error {
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	return nil
}
