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
	active, err := loadedRunner(settings)
	if err != nil {
		return nil, fmt.Errorf("New: %w", err)
	}

	return coverageAnalyzer(active), nil
}

func (runResult) isAnalyzerResult() {}

func coverageAnalyzer(active *runner) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: Name,
		Doc:  Doc,
		Run: func(pass *analysis.Pass) (any, error) {
			return runRunner(active, pass)
		},
		ResultType: reflect.TypeFor[runResult](),
	}
}

func decodeSettings(settings *Settings, data []byte) error {
	type settingsAlias Settings

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var alias settingsAlias

	err := decoder.Decode(&alias)
	if err != nil {
		return fmt.Errorf(errUnmarshalSettings, err)
	}

	*settings = Settings(alias)

	return nil
}

func decodeUnmarshaledSettings(settings *Settings, data []byte) error {
	remapped, err := remapKebabKeys(data)
	if err != nil {
		return fmt.Errorf(errUnmarshalSettings, err)
	}

	err = decodeSettings(settings, remapped)
	if err != nil {
		return fmt.Errorf(errUnmarshalSettings, err)
	}

	return nil
}

func diagnosticFileMatches(pass *analysis.Pass, file *ast.File, wanted string) bool {
	if file == nil {
		return false
	}

	position := pass.Fset.PositionFor(file.Package, true)

	return normalizeFilename(position.Filename) == wanted
}

func diagnosticPosition(pass *analysis.Pass, resultFile string) token.Pos {
	if pass.Fset == nil || resultFile == emptyString {
		return firstFilePosition(pass.Files)
	}

	position := matchingDiagnosticPosition(pass, resultFile)

	if position.IsValid() {
		return position
	}

	return firstFilePosition(pass.Files)
}

func doLoad(active *runner) {
	run, err := coverlint.Check(context.Background(), active.config)
	if err != nil {
		active.loadErr = fmt.Errorf("run %s: %w", Name, err)

		return
	}

	recordViolations(active, &run)
}

func emptyResult() runResult {
	return runResult{}
}

func firstFilePosition(files []*ast.File) token.Pos {
	for i := range files {
		if files[i] != nil {
			return files[i].Package
		}
	}

	return token.NoPos
}

func loadRunner(active *runner) error {
	active.loadOnce.Do(func() { doLoad(active) })

	if active.loadErr != nil {
		return fmt.Errorf(errLoad, active.loadErr)
	}

	return nil
}

func loadedRunner(settings *Settings) (*runner, error) {
	active := newRunner(settings)

	err := loadRunner(active)
	if err != nil {
		return nil, fmt.Errorf(errLoad, err)
	}

	return active, nil
}

func matchingDiagnosticPosition(pass *analysis.Pass, resultFile string) token.Pos {
	wanted := normalizeFilename(resultFile)

	for i := range pass.Files {
		if diagnosticFileMatches(pass, pass.Files[i], wanted) {
			return pass.Files[i].Package
		}
	}

	return token.NoPos
}

func newRunner(settings *Settings) *runner {
	return &runner{
		config:     settingsToConfig(settings),
		violations: make(map[string]coverlint.Result),
	}
}

func normalizeFilename(filename string) string {
	absolute, err := filepath.Abs(filename)
	if err == nil {
		filename = absolute
	}

	return filepath.Clean(filename)
}

func recordViolations(active *runner, run *coverlint.Run) {
	active.violations = make(map[string]coverlint.Result, run.Report.Failed)

	for i := range run.Report.Results {
		if run.Report.Results[i].Violation {
			active.violations[run.Report.Results[i].ImportPath] = run.Report.Results[i]
		}
	}
}

func remapKebabKeys(data []byte) ([]byte, error) {
	marshaled, err := remapKebabKeysWith(data, json.Marshal)
	if err != nil {
		return nil, fmt.Errorf(errRemapKebabKeys, err)
	}

	return marshaled, nil
}

func remapKebabKeysWith(data []byte, marshal func(any) ([]byte, error)) ([]byte, error) {
	var raw map[string]json.RawMessage

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf(errRemapKebabKeys, err)
	}

	remapTestArgsAlias(raw, legacyTestArgsKeyName)
	remapTestArgsAlias(raw, camelTestArgsKeyName)

	marshaled, err := marshal(raw)
	if err != nil {
		return nil, fmt.Errorf(errRemapKebabKeys, err)
	}

	return marshaled, nil
}

func remapTestArgsAlias(raw map[string]json.RawMessage, alias string) {
	value, ok := raw[alias]

	if !ok {
		return
	}

	delete(raw, alias)

	if _, hasCanonical := raw[testArgsKeyName]; hasCanonical {
		return
	}

	raw[testArgsKeyName] = value
}

func reportPass(active *runner, pass *analysis.Pass) runResult {
	if pass.Pkg == nil || pass.Pkg.Path() == emptyString {
		return emptyResult()
	}

	result, ok := active.violations[pass.Pkg.Path()]

	if !ok {
		return emptyResult()
	}

	return reportViolation(active, pass, &result)
}

func reportViolation(active *runner, pass *analysis.Pass, result *coverlint.Result) runResult {
	existing, alreadyReported := active.reported.LoadOrStore(result.ImportPath, struct{}{})

	if alreadyReported || existing == nil {
		return emptyResult()
	}

	pass.Report(analysis.Diagnostic{
		Pos:            diagnosticPosition(pass, result.File),
		End:            token.NoPos,
		Category:       Name,
		Message:        result.Message,
		URL:            emptyString,
		SuggestedFixes: nil,
		Related:        nil,
	})

	return emptyResult()
}

func runRunner(active *runner, pass *analysis.Pass) (runResult, error) {
	err := loadRunner(active)
	if err != nil {
		return emptyResult(), fmt.Errorf(errRun, err)
	}

	return reportPass(active, pass), nil
}

func settingsToConfig(settings *Settings) *coverlint.Config {
	if settings == nil {
		return &coverlint.Config{}
	}

	return &coverlint.Config{
		Rules:    settings.Rules,
		Exclude:  settings.Exclude,
		Packages: settings.Packages,
		Timeout:  settings.Timeout,
		TestArgs: settings.TestArgs,
	}
}

// UnmarshalSettings accepts the documented legacy test-args key and the
// camelCase key so DisallowUnknownFields still applies; both remap to test_args.
func UnmarshalSettings(data []byte, settings *Settings) error {
	err := decodeUnmarshaledSettings(settings, data)
	if err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}

	return nil
}
