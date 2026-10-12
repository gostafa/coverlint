// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"github.com/cenkalti/backoff/v7"
	"github.com/gostafa/singleton"
)

// Run retries only allowlisted transient system errors. Each attempt must release
// resources on failure. The elapsed budget limits scheduling, not in-flight I/O.
// Dependency-specific errors are unwrapped before returning to the caller.
func Run(ctx context.Context, operation func() error) error {
	completed, err := backoff.Retry(ctx, func() (bool, error) {
		return attempt(ctx, operation)
	}, backoff.WithBackOff(policy()), backoff.WithMaxTries(maxAttempts),
		backoff.WithMaxElapsedTime(elapsedBudget))

	if !completed {
		return fmt.Errorf("retry operation: %w", operationError(ctx, err))
	}

	return nil
}

// Stop marks an unsafe-to-repeat failure as permanent. The singleton marker is
// translated to the retry engine's marker and stripped at the Run boundary.
func Stop(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("permanent operation: %w", singleton.Permanent(err))
}

func attempt(ctx context.Context, operation func() error) (bool, error) {
	cause := context.Cause(ctx)
	if cause != nil {
		return false, fmt.Errorf("operation context: %w", backoff.Permanent(cause))
	}

	err := operation()
	if err != nil {
		return false, fmt.Errorf("operation attempt: %w", classify(err))
	}

	return true, nil
}

func classify(err error) error {
	if failure, ok := errors.AsType[*singleton.PermanentError](err); ok {
		return fmt.Errorf("stopped operation: %w", backoff.Permanent(failure.Err))
	}

	if !transient(err) {
		return fmt.Errorf("permanent failure: %w", backoff.Permanent(err))
	}

	return err
}

func operationError(ctx context.Context, err error) error {
	cause := context.Cause(ctx)
	if cause != nil {
		return fmt.Errorf(
			"operation canceled: %w",
			errors.Join(cause, backoff.AsRetryError(err).LastErr),
		)
	}

	return fmt.Errorf("operation failed: %w", backoff.AsRetryError(err).LastErr)
}

func policy() *backoff.ExponentialBackOff {
	settings := backoff.NewExponentialBackOff()

	settings.InitialInterval = initialInterval
	settings.MaxInterval = maximumInterval
	settings.Multiplier = multiplier
	settings.RandomizationFactor = jitter

	return settings
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
