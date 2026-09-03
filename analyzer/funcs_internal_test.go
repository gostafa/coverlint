// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/coverlint"
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

	file, err := parser.ParseFile(fileSet, "helpers.go", "package helpers\n", parser.PackageClauseOnly)
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

	if got := diagnosticPosition(&analysis.Pass{Fset: nil, Files: []*ast.File{file}}, ""); got != file.Package {
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

	err := loadRunner(active)
	if err == nil || !strings.Contains(err.Error(), "load:") {
		t.Fatalf("loadRunner error = %v", err)
	}

	_, err = loadedRunner(&Settings{
		Timeout:  "1m",
		TestArgs: []string{"-covermode=atomic"},
	})
	if err == nil || !strings.Contains(err.Error(), "load:") {
		t.Fatalf("loadedRunner error = %v", err)
	}

	cached := &runner{loadErr: errBoom}
	cached.loadOnce.Do(func() {})

	_, err = runRunner(cached, &analysis.Pass{})
	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("runRunner error = %v, want boom", err)
	}
}

func TestRemapKebabKeysFailures(t *testing.T) {
	_, err := remapKebabKeys([]byte(`[`))
	if err == nil || !strings.Contains(err.Error(), "remapKebabKeys:") {
		t.Fatalf("unmarshal error = %v", err)
	}

	original := jsonMarshal
	t.Cleanup(func() { jsonMarshal = original })

	jsonMarshal = func(any) ([]byte, error) { return nil, errBoom }

	_, err = remapKebabKeys([]byte(`{"timeout":"1m"}`))
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

func TestDoLoadRecordsError(t *testing.T) {
	t.Parallel()

	active := &runner{
		config: &coverlint.Config{
			Timeout:  "1m",
			TestArgs: []string{"-covermode=atomic"},
		},
		violations: map[string]coverlint.Result{},
	}

	doLoad(active)

	if active.loadErr == nil || !strings.Contains(active.loadErr.Error(), "run coverlint:") {
		t.Fatalf("loadErr = %v", active.loadErr)
	}
}
