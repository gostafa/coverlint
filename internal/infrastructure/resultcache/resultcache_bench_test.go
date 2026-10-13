// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package resultcache

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkCachedResult(b *testing.B) {
	b.Run("singleton", func(b *testing.B) {
		get := New(func(context.Context) (int, error) { return 42, nil })
		ctx := b.Context()
		if _, err := get(ctx); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if value, err := get(ctx); value != 42 || err != nil {
				b.Fatalf("result = %d, %v", value, err)
			}
		}
	})
	b.Run("sync.Once", func(b *testing.B) {
		var once sync.Once
		var value int
		initialize := func() { value = 42 }
		once.Do(initialize)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			once.Do(initialize)
			if value != 42 {
				b.Fatalf("result = %d", value)
			}
		}
	})
}

func BenchmarkInitialization(b *testing.B) {
	b.Run("singleton", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			get := New(func(context.Context) (int, error) { return 42, nil })
			if value, err := get(b.Context()); value != 42 || err != nil {
				b.Fatalf("result = %d, %v", value, err)
			}
		}
	})
	b.Run("sync.Once", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var once sync.Once
			var value int
			once.Do(func() { value = 42 })
			if value != 42 {
				b.Fatalf("result = %d", value)
			}
		}
	})
}
