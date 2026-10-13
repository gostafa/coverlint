// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package resultcache

import (
	"context"
	"errors"
	"fmt"

	"github.com/gostafa/coverlint/internal/features/policy/domain"
	reportingports "github.com/gostafa/coverlint/internal/features/reporting/ports"
	"github.com/gostafa/singleton"
)

// New caches one factory result or failure. Caller cancellation only ends that
// caller's wait. The factory owns its timeout and is never retried. Callers must
// treat the returned value as immutable.
func New[Value any](
	factory func(context.Context) (Value, error),
) func(context.Context) (Value, error) {
	var provider valueProvider[Value] = singleton.MustNew(factory,
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

// NewProvider caches reporting results through the feature-owned result-provider port.
func NewProvider(factory func(context.Context) (map[string]domain.Result, error)) *Adapter {
	return &Adapter{get: New(factory)}
}

func cacheError(err error) error {
	if failure, ok := errors.AsType[*singleton.InitError](err); ok {
		return fmt.Errorf("initialize result cache: %w", failure.Err())
	}

	return fmt.Errorf("wait for result cache: %w", err)
}

// Get returns the cached result, preserving per-caller cancellation.
func (adapter *Adapter) Get(ctx context.Context) (reportingports.Snapshot, error) {
	value, err := adapter.get(ctx)

	return reportingports.Snapshot{Violations: value}, errors.Join(err)
}
