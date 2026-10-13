// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/infrastructure/resultcache"
	"golang.org/x/tools/go/analysis"
)

var errBoom = errors.New("boom")

func TestIsAnalyzerResult(t *testing.T) {
	t.Parallel()

	var result analyzerResult = runResult{}

	result.isAnalyzerResult()
}

func TestNewReturnsLoadError(t *testing.T) {
	t.Parallel()

	_, err := New(&Settings{
		Timeout:  "1m",
		TestArgs: []string{"-covermode=atomic"},
	})
	if err == nil || !strings.Contains(err.Error(), "New:") {
		t.Fatalf("error = %v, want New-wrapped load error", err)
	}
}

func TestDecodeUnmarshaledSettingsRemapError(t *testing.T) {
	t.Parallel()

	var settings Settings

	err := decodeUnmarshaledSettings(&settings, []byte(`not-json`))
	if err == nil || !strings.Contains(err.Error(), "UnmarshalSettings:") {
		t.Fatalf("error = %v, want remap failure", err)
	}
}

func TestDiagnosticHelpers(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(
		fileSet,
		"helpers.go",
		"package helpers\n",
		parser.PackageClauseOnly,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fileSet,
		Files: []*ast.File{nil, file},
		Pkg:   types.NewPackage("example.com/helpers", "helpers"),
		Report: func(analysis.Diagnostic) {
		},
	}

	if diagnosticFileMatches(pass, nil, "helpers.go") {
		t.Fatal("nil file matched")
	}

	if got := firstFilePosition(nil); got != token.NoPos {
		t.Fatalf("firstFilePosition(nil) = %v, want NoPos", got)
	}

	if got := firstFilePosition([]*ast.File{nil, nil}); got != token.NoPos {
		t.Fatalf("firstFilePosition(all nil) = %v, want NoPos", got)
	}

	if got := firstFilePosition([]*ast.File{nil, file}); got != file.Package {
		t.Fatalf("firstFilePosition = %v, want %v", got, file.Package)
	}

	if got := diagnosticPosition(
		&analysis.Pass{Fset: nil, Files: []*ast.File{file}},
		"",
	); got != file.Package {
		t.Fatalf("nil Fset position = %v, want first file", got)
	}

	if got := diagnosticPosition(pass, ""); got != file.Package {
		t.Fatalf("empty result file position = %v, want first file", got)
	}

	if got := matchingDiagnosticPosition(pass, "missing.go"); got != token.NoPos {
		t.Fatalf("matchingDiagnosticPosition = %v, want NoPos", got)
	}

	if got := diagnosticPosition(pass, "missing.go"); got != file.Package {
		t.Fatalf("fallback position = %v, want first file", got)
	}
}

func TestLoadAndRunRunnerErrors(t *testing.T) {
	t.Parallel()

	active := newRunner(&Settings{
		Timeout:  "1m",
		TestArgs: []string{"-covermode=atomic"},
	})

	_, err := loadRunner(active)
	if err == nil || !strings.Contains(err.Error(), "load:") {
		t.Fatalf("loadRunner error = %v", err)
	}

	cached := &runner{
		load: resultcache.New(func(context.Context) (map[string]coverlint.Result, error) {
			return nil, errBoom
		}),
	}

	_, err = loadRunner(cached)
	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("loadRunner error = %v, want boom", err)
	}
}

func TestRemapKebabKeysFailures(t *testing.T) {
	_, err := remapKebabKeys([]byte(`[`))
	if err == nil || !strings.Contains(err.Error(), "remapKebabKeys:") {
		t.Fatalf("unmarshal error = %v", err)
	}

	_, err = remapKebabKeysWith([]byte(`{"timeout":"1m"}`), func(any) ([]byte, error) {
		return nil, errBoom
	})
	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("marshal error = %v, want boom", err)
	}
}

func TestRemapTestArgsAliasKeepsCanonical(t *testing.T) {
	t.Parallel()

	var settings Settings

	err := UnmarshalSettings(
		[]byte(`{"test_args":["-race"],"test-args":["-count=1"],"testArgs":["-v"]}`),
		&settings,
	)
	if err != nil {
		t.Fatalf("UnmarshalSettings: %v", err)
	}

	if len(settings.TestArgs) != 1 || settings.TestArgs[0] != "-race" {
		t.Fatalf("TestArgs = %#v, want canonical value", settings.TestArgs)
	}
}

func TestSettingsToConfigNil(t *testing.T) {
	t.Parallel()

	cfg := settingsToConfig(nil)
	if cfg == nil {
		t.Fatal("settingsToConfig(nil) returned nil")
	}

	if cfg.Timeout != "" || len(cfg.Rules) != 0 || len(cfg.Packages) != 0 {
		t.Fatalf("config = %#v, want empty defaults", cfg)
	}
}

func TestSettingsToConfigCopiesResultPaths(t *testing.T) {
	t.Parallel()

	cfg := settingsToConfig(&Settings{
		TestResultPath:     "/tmp/test.txt",
		CoverageResultPath: "/tmp/c.out",
	})

	if cfg.TestResultPath != "/tmp/test.txt" {
		t.Fatalf("TestResultPath = %q", cfg.TestResultPath)
	}

	if cfg.CoverageResultPath != "/tmp/c.out" {
		t.Fatalf("CoverageResultPath = %q", cfg.CoverageResultPath)
	}
}

func TestDoLoadRecordsError(t *testing.T) {
	t.Parallel()

	_, err := doLoad(t.Context(), &coverlint.Config{
		Timeout:  "1m",
		TestArgs: []string{"-covermode=atomic"},
	})

	if err == nil || !strings.Contains(err.Error(), "run coverlint:") {
		t.Fatalf("loadErr = %v", err)
	}
}

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
