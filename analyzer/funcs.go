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
	"slices"

	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/infrastructure/resultcache"
	"golang.org/x/tools/go/analysis"
)

// New returns a go/analysis Analyzer that eagerly runs the coverage check.
func New(settings *Settings) (*analysis.Analyzer, error) {
	active := newRunner(settings)

	violations, err := loadRunner(active)
	if err != nil {
		return nil, fmt.Errorf("New: %w", err)
	}

	return coverageAnalyzer(active, violations), nil
}

// UnmarshalSettings accepts snake_case keys plus documented kebab/camel aliases.
func UnmarshalSettings(data []byte, settings *Settings) error {
	err := decodeUnmarshaledSettings(settings, data)
	if err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}

	return nil
}

func applyKebabKeyAliases(raw map[string]json.RawMessage) {
	remapKeyAlias(raw, legacyTestArgsKeyName, testArgsKeyName)
	remapKeyAlias(raw, camelTestArgsKeyName, testArgsKeyName)
	remapKeyAlias(raw, legacyTestResultPathKeyName, testResultPathKeyName)
	remapKeyAlias(raw, legacyCoverageResultPathKeyName, coverageResultPathKeyName)
}

func coverageAnalyzer(active *runner, violations map[string]coverlint.Result) *analysis.Analyzer {
	instance := new(analysis.Analyzer)

	instance.Name = Name
	instance.Doc = Doc
	instance.Run = func(pass *analysis.Pass) (any, error) {
		return reportPass(active, pass, violations), nil
	}
	instance.ResultType = reflect.TypeFor[runResult]()

	return instance
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

func doLoad(ctx context.Context, config *coverlint.Config) (map[string]coverlint.Result, error) {
	run, err := coverlint.Check(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", Name, err)
	}

	return recordViolations(&run), nil
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

func loadRunner(active *runner) (map[string]coverlint.Result, error) {
	violations, err := active.load(context.Background())
	if err != nil {
		return nil, fmt.Errorf(errLoad, err)
	}

	return violations, nil
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
	config := settingsToConfig(settings)

	active := new(runner)

	active.load = resultcache.New(func(ctx context.Context) (map[string]coverlint.Result, error) {
		return doLoad(ctx, config)
	})

	return active
}

func normalizeFilename(filename string) string {
	absolute, err := filepath.Abs(filename)
	if err == nil {
		filename = absolute
	}

	return filepath.Clean(filename)
}

func recordViolations(run *coverlint.Run) map[string]coverlint.Result {
	violations := make(map[string]coverlint.Result, run.Report.Failed)

	for i := range run.Report.Results {
		if run.Report.Results[i].Violation {
			violations[run.Report.Results[i].ImportPath] = run.Report.Results[i]
		}
	}

	return violations
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

	applyKebabKeyAliases(raw)

	marshaled, err := marshal(raw)
	if err != nil {
		return nil, fmt.Errorf(errRemapKebabKeys, err)
	}

	return marshaled, nil
}

func remapKeyAlias(raw map[string]json.RawMessage, alias, canonical string) {
	value, ok := raw[alias]

	if !ok {
		return
	}

	delete(raw, alias)

	if _, hasCanonical := raw[canonical]; hasCanonical {
		return
	}

	raw[canonical] = value
}

func reportPass(
	active *runner,
	pass *analysis.Pass,
	violations map[string]coverlint.Result,
) runResult {
	if pass.Pkg == nil || pass.Pkg.Path() == emptyString {
		return emptyResult()
	}

	result, ok := violations[pass.Pkg.Path()]

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

func settingsToConfig(settings *Settings) *coverlint.Config {
	if settings == nil {
		return new(coverlint.Config)
	}

	return &coverlint.Config{
		Rules:              slices.Clone(settings.Rules),
		Packages:           slices.Clone(settings.Packages),
		Timeout:            settings.Timeout,
		TestArgs:           slices.Clone(settings.TestArgs),
		TestResultPath:     settings.TestResultPath,
		CoverageResultPath: settings.CoverageResultPath,
	}
}

func (runResult) isAnalyzerResult() {}
