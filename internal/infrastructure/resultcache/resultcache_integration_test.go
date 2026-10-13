// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package resultcache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cenkalti/backoff/v7"
	"github.com/gostafa/coverlint/internal/features/policy/domain"
	reportingports "github.com/gostafa/coverlint/internal/features/reporting/ports"
	"github.com/gostafa/coverlint/internal/infrastructure/resultcache"
	"github.com/gostafa/singleton"
)

func TestReportingProviderSharesResultsThroughPort(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	var provider reportingports.ResultProvider = resultcache.NewProvider(
		func(context.Context) (map[string]domain.Result, error) {
			calls.Add(1)
			return map[string]domain.Result{
				"example.com/pkg": {Message: "below minimum", Violation: true},
			}, nil
		},
	)
	for range 2 {
		snapshot, err := provider.Get(t.Context())
		if err != nil || len(snapshot.Violations) != 1 ||
			snapshot.Violations["example.com/pkg"].Message != "below minimum" {
			t.Fatalf("snapshot = %#v, error = %v", snapshot, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("factory calls = %d, want 1", calls.Load())
	}
}

func TestSharedResultAndCachedFailure(t *testing.T) {
	t.Parallel()
	for _, failing := range []bool{false, true} {
		var calls atomic.Int32
		sentinel := errors.New("factory failed")
		get := resultcache.New(func(context.Context) (*int, error) {
			calls.Add(1)
			if failing {
				return nil, sentinel
			}
			value := 42
			return &value, nil
		})
		var workers sync.WaitGroup
		values := make([]*int, 32)
		for index := range values {
			workers.Go(func() {
				value, err := get(t.Context())
				if failing {
					var initErr *singleton.InitError
					var retryErr *backoff.RetryError
					if !errors.Is(err, sentinel) || value != nil || errors.As(err, &initErr) ||
						errors.As(err, &retryErr) {
						t.Errorf("failure = %v, %v", value, err)
					}
				} else if err != nil || value == nil || *value != 42 {
					t.Errorf("result = %v, %v", value, err)
				}
				values[index] = value
			})
		}
		workers.Wait()
		if calls.Load() != 1 {
			t.Fatalf("factory calls = %d, want 1", calls.Load())
		}
		for _, value := range values {
			if value != values[0] {
				t.Fatal("callers received different shared values")
			}
		}
	}
}

func TestCancellationDoesNotPoisonInitialization(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	get := resultcache.New(func(ctx context.Context) (int, error) {
		close(started)
		<-release
		if ctx.Err() != nil {
			t.Errorf("caller canceled factory: %v", ctx.Err())
		}
		if _, deadline := ctx.Deadline(); deadline {
			t.Error("shared initialization acquired a deadline")
		}
		return 42, nil
	})
	ctx, cancel := context.WithCancelCause(t.Context())
	sentinel := errors.New("caller gave up")
	done := make(chan error, 1)
	go func() {
		_, err := get(ctx)
		done <- err
	}()
	<-started
	cancel(sentinel)
	if err := <-done; !errors.Is(err, sentinel) {
		t.Errorf("cancellation = %v", err)
	}
	close(release)
	if value, err := get(t.Context()); value != 42 || err != nil {
		t.Fatalf("later caller = %d, %v", value, err)
	}
}

func TestFactoryPanicIsReplayed(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	get := resultcache.New(func(context.Context) (int, error) {
		calls.Add(1)
		panic("factory panic")
	})
	for range 2 {
		func() {
			defer func() {
				if value := recover(); value != "factory panic" {
					t.Errorf("panic = %v", value)
				}
			}()
			_, _ = get(t.Context())
		}()
	}
	if calls.Load() != 1 {
		t.Fatalf("panicking factory calls = %d", calls.Load())
	}
}

func TestProvidersRemainIndependent(t *testing.T) {
	t.Parallel()
	first := resultcache.New(func(context.Context) (string, error) { return "first", nil })
	second := resultcache.New(func(context.Context) (string, error) { return "second", nil })
	firstValue, firstErr := first(t.Context())
	secondValue, secondErr := second(t.Context())
	if firstValue != "first" || secondValue != "second" || firstErr != nil || secondErr != nil {
		t.Fatalf("providers = %q/%q, %v/%v", firstValue, secondValue, firstErr, secondErr)
	}
}
