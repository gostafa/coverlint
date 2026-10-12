// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"context"
	"go/types"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/coverage/adapters/resultcache"
	"golang.org/x/tools/go/analysis"
)

func TestSharedSnapshotAndConcurrentDiagnosticDeduplication(t *testing.T) {
	t.Parallel()
	var loads, reports atomic.Int32
	active := &runner{
		load: resultcache.New(func(context.Context) (map[string]coverlint.Result, error) {
			loads.Add(1)
			return recordViolations(
				&coverlint.Run{Report: coverlint.Report{Failed: 1, Results: []coverlint.Result{
					{ImportPath: "example.com/failing", Message: "below minimum", Violation: true},
					{ImportPath: "example.com/passing", Violation: false},
				}}},
			), nil
		}),
	}
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			violations, err := loadRunner(active)
			if err != nil {
				t.Error(err)
				return
			}
			analyzer := coverageAnalyzer(active, violations)
			pass := &analysis.Pass{
				Pkg:    types.NewPackage("example.com/failing", "failing"),
				Report: func(analysis.Diagnostic) { reports.Add(1) },
			}
			if _, err := analyzer.Run(pass); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if loads.Load() != 1 || reports.Load() != 1 {
		t.Fatalf("loads %d reports %d, want 1 each", loads.Load(), reports.Load())
	}
}

func TestSettingsSnapshotOwnsSlices(t *testing.T) {
	t.Parallel()
	settings := &Settings{
		Rules:    []coverlint.Rule{{Pattern: "**", Min: 0.8}},
		Packages: []string{"./old"}, TestArgs: []string{"-race"},
	}
	snapshot := settingsToConfig(settings)
	settings.Rules[0].Min = 0.1
	settings.Packages[0] = "./new"
	settings.TestArgs[0] = "-count=0"
	if snapshot.Rules[0].Min != 0.8 || snapshot.Packages[0] != "./old" ||
		snapshot.TestArgs[0] != "-race" {
		t.Fatalf("settings escaped into snapshot: %#v", snapshot)
	}
	active := newRunner(&Settings{TestArgs: []string{"-covermode=atomic"}})
	if _, err := loadRunner(active); err == nil {
		t.Fatal("expected validation error")
	}
}
