// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"go/types"
	"sync"
	"testing"

	"github.com/gostafa/coverlint/coverlint"
	"golang.org/x/tools/go/analysis"
)

func BenchmarkAnalyzerSnapshot(b *testing.B) {
	for _, cached := range []bool{true, false} {
		name := "snapshot"
		if !cached {
			name = "sync.Once"
		}
		b.Run(name, func(b *testing.B) {
			active := &runner{}
			violations := map[string]coverlint.Result{}
			pass := &analysis.Pass{Pkg: types.NewPackage("example.com/passing", "passing")}
			var once sync.Once
			once.Do(func() {})
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if !cached {
					once.Do(func() {})
				}
				reportPass(active, pass, violations)
			}
		})
	}
}
