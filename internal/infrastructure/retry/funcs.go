// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gostafa/singleton"
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

// Run retries only allowlisted transient system errors. Each attempt must release
// resources on failure. The elapsed budget limits scheduling, not in-flight I/O.
// Dependency-specific errors are unwrapped before returning to the caller.
func Run(ctx context.Context, operation func() error) error {
	err := runAttempt(ctx, &retryAttempt{
		operation: operation, started: time.Now(), attempts: zero,
	})
	if err != nil {
		return fmt.Errorf("retry operation: %w", err)
	}

	return nil
}

// Stop marks an unsafe-to-repeat failure as permanent. The singleton marker is
// stripped at the Run boundary.
func Stop(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("permanent operation: %w", singleton.Permanent(err))
}

func attempt(ctx context.Context, operation func() error) error {
	cause := context.Cause(ctx)
	if cause != nil {
		return fmt.Errorf("operation context: %w", Stop(cause))
	}

	return errors.Join(operation())
}

func classify(err error) (bool, error) {
	if failure, ok := errors.AsType[*singleton.PermanentError](err); ok {
		return false, fmt.Errorf("stopped operation: %w", failure.Err)
	}

	if !transient(err) {
		return false, fmt.Errorf("permanent failure: %w", err)
	}

	return true, err
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

func repeatAttempt(ctx context.Context, active *retryAttempt, cause error) error {
	delay, allowed := retrySchedule(active)

	if !allowed {
		return fmt.Errorf("retry stopped: %w", errors.Join(cause, context.Cause(ctx)))
	}

	err := waitRetry(ctx, delay)
	if err != nil {
		return fmt.Errorf("retry canceled: %w", errors.Join(cause, err))
	}

	return errors.Join(runAttempt(ctx, active))
}

func retryDelay(attempts int) time.Duration {
	interval := min(
		time.Duration(float64(initialInterval)*math.Pow(multiplier, float64(attempts))),
		maximumInterval,
	)

	return time.Duration(float64(interval) * (one - jitter + retryJitter()*jitterRange))
}

func retryJitter() float64 {
	text := rand.Text()
	sample := float64(zero)

	for index := range jitterDigits {
		sample = sample*float64(
			len(jitterAlphabet),
		) + float64(
			strings.IndexByte(jitterAlphabet, text[index]),
		)
	}

	return sample / math.Pow(float64(len(jitterAlphabet)), jitterDigits)
}

func retrySchedule(active *retryAttempt) (time.Duration, bool) {
	delay := retryDelay(active.attempts - one)

	return delay, active.attempts < maxAttempts && time.Since(active.started)+delay <= elapsedBudget
}

func runAttempt(ctx context.Context, active *retryAttempt) error {
	active.attempts++

	err := attempt(ctx, active.operation)
	if err == nil {
		return nil
	}

	shouldRetry, cause := classify(err)

	if !shouldRetry {
		return fmt.Errorf("operation failed: %w", errors.Join(cause, context.Cause(ctx)))
	}

	return errors.Join(repeatAttempt(ctx, active, cause))
}

func transient(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	codes := []syscall.Errno{
		syscall.EINTR,
		syscall.EAGAIN,
		syscall.EMFILE,
		syscall.ENFILE,
		syscall.ETIMEDOUT,
	}

	for index := range codes {
		if errors.Is(err, codes[index]) {
			return true
		}
	}

	return false
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)

	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("retry wait: %w", context.Cause(ctx))
	case <-timer.C:
		return nil
	}
}

func wrapCreateError(err error) error {
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	return nil
}
