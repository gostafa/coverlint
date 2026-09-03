// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"reflect"

	"github.com/gostafa/coverlint/coverlint"
	"golang.org/x/tools/go/analysis"
)

// New returns a go/analysis Analyzer that eagerly runs the coverage check.
func New(settings *Settings) (*analysis.Analyzer, error) {
	active := &runner{
		config:     settingsToConfig(settings),
		violations: make(map[string]coverlint.Result),
	}

	err := active.load()
	if err != nil {
		return nil, err
	}

	return &analysis.Analyzer{
		Name:       Name,
		Doc:        Doc,
		Run:        func(pass *analysis.Pass) (any, error) { return active.runPass(pass) },
		ResultType: reflect.TypeFor[runResult](),
	}, nil
}

func (r *runner) load() error {
	r.loadOnce.Do(func() {
		run, err := coverlint.Check(context.Background(), r.config)
		if err != nil {
			r.loadErr = fmt.Errorf("run %s: %w", Name, err)

			return
		}

		r.violations = make(map[string]coverlint.Result, run.Report.Failed)

		for _, result := range run.Report.Results {
			if result.Violation {
				r.violations[result.ImportPath] = result
			}
		}
	})

	return r.loadErr
}

// UnmarshalSettings accepts the documented legacy test-args key and the
// camelCase key so DisallowUnknownFields still applies.
func UnmarshalSettings(data []byte, settings *Settings) error {
	err := decodeUnmarshaledSettings(settings, data)
	if err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}

	return nil
}

func decodeUnmarshaledSettings(settings *Settings, data []byte) error {
	remapped, err := remapKebabKeys(data)
	if err != nil {
		return fmt.Errorf("UnmarshalSettings: %w", err)
	}

	err = decodeSettings(settings, remapped)
	if err != nil {
		return fmt.Errorf("UnmarshalSettings: %w", err)
	}

	return nil
}

func decodeSettings(settings *Settings, data []byte) error {
	type settingsAlias Settings

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var alias settingsAlias

	err := decoder.Decode(&alias)
	if err != nil {
		return fmt.Errorf("UnmarshalSettings: %w", err)
	}

	*settings = Settings(alias)

	return nil
}

func remapKebabKeys(data []byte) ([]byte, error) {
	var raw map[string]json.RawMessage

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("remapKebabKeys: %w", err)
	}

	if value, ok := raw["test-args"]; ok {
		if _, has := raw["testArgs"]; !has {
			raw["testArgs"] = value
		}

		delete(raw, "test-args")
	}

	return json.Marshal(raw)
}

func settingsToConfig(settings *Settings) coverlint.Config {
	if settings == nil {
		return coverlint.Config{}
	}

	return coverlint.Config{
		Min:       settings.Min,
		Overrides: settings.Overrides,
		Exclude:   settings.Exclude,
		Packages:  settings.Packages,
		Timeout:   settings.Timeout,
		TestArgs:  settings.TestArgs,
	}
}

func (r *runner) runPass(pass *analysis.Pass) (any, error) {
	if err := r.load(); err != nil {
		return nil, err
	}

	if pass.Pkg == nil || pass.Pkg.Path() == "" {
		return emptyResult(), nil
	}

	result, ok := r.violations[pass.Pkg.Path()]

	if !ok {
		return emptyResult(), nil
	}

	if _, loaded := r.reported.LoadOrStore(result.ImportPath, struct{}{}); loaded {
		return emptyResult(), nil
	}

	pass.Report(analysis.Diagnostic{
		Pos:            diagnosticPosition(pass, result.File),
		End:            token.NoPos,
		Category:       Name,
		Message:        result.Message,
		URL:            "",
		SuggestedFixes: nil,
		Related:        nil,
	})

	return emptyResult(), nil
}

func emptyResult() any {
	return runResult{}
}

func diagnosticPosition(pass *analysis.Pass, resultFile string) token.Pos {
	if pass.Fset != nil && resultFile != "" {
		position := matchingDiagnosticPosition(pass, resultFile)

		if position.IsValid() {
			return position
		}
	}

	return firstFilePosition(pass.Files)
}

func matchingDiagnosticPosition(pass *analysis.Pass, resultFile string) token.Pos {
	wanted := normalizeFilename(resultFile)

	for _, file := range pass.Files {
		if diagnosticFileMatches(pass, file, wanted) {
			return file.Package
		}
	}

	return token.NoPos
}

func diagnosticFileMatches(pass *analysis.Pass, file *ast.File, wanted string) bool {
	if file == nil {
		return false
	}

	position := pass.Fset.PositionFor(file.Package, true)

	return normalizeFilename(position.Filename) == wanted
}

func firstFilePosition(files []*ast.File) token.Pos {
	for _, file := range files {
		if file != nil {
			return file.Package
		}
	}

	return token.NoPos
}

func normalizeFilename(filename string) string {
	absolute, err := filepath.Abs(filename)
	if err == nil {
		filename = absolute
	}

	return filepath.Clean(filename)
}
