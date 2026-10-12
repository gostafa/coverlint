// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package resultcache

import (
	"context"
	"errors"
	"fmt"

	"github.com/gostafa/singleton"
)

// New caches one factory result or failure. Caller cancellation only ends that
// caller's wait. The factory owns its timeout and is never retried. Callers must
// treat the returned value as immutable.
func New[Value any](
	factory func(context.Context) (Value, error),
) func(context.Context) (Value, error) {
	provider := singleton.MustNew(factory,
		singleton.WithMaxAttempts(cacheMaxAttempts),
		singleton.WithInitializationTimeout(unlimitedInitialization),
	)

	return func(ctx context.Context) (Value, error) {
		value, err := provider.Get(ctx)
		if err != nil {
			return value, cacheError(err)
		}

		return value, nil
	}
}

func cacheError(err error) error {
	if failure, ok := errors.AsType[*singleton.InitError](err); ok {
		return fmt.Errorf("initialize result cache: %w", failure.Err())
	}

	return fmt.Errorf("wait for result cache: %w", err)
}
