// Package coverlint registers the golangci-lint coverlint plugin.
package coverlint

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/golangci/plugin-module-register/register"
	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
	"golang.org/x/tools/go/analysis"
)

var _ = func() bool {
	register.Plugin(
		coveragefeature.Name,
		func(rawSettings any) (register.LinterPlugin, error) {
			return newPlugin(rawSettings)
		},
	)

	return coveragefeature.Name != ""
}()

type plugin struct {
	config     coveragefeature.Config
	loadOnce   sync.Once
	loadErr    error
	violations map[string]coveragefeature.Result
	reported   sync.Map
}

var _ register.LinterPlugin = (*plugin)(nil)

func newPlugin(rawSettings any) (*plugin, error) {
	var config coveragefeature.Config

	if rawSettings != nil {
		decoded, err := register.DecodeSettings[coveragefeature.Config](rawSettings)
		if err != nil {
			return nil, fmt.Errorf("decode %s settings: %w", coveragefeature.Name, err)
		}

		config = decoded
	}

	return &plugin{
		config:     config,
		loadOnce:   sync.Once{},
		loadErr:    nil,
		violations: make(map[string]coveragefeature.Result),
		reported:   sync.Map{},
	}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	p.loadOnce.Do(func() {
		run, err := coveragefeature.Check(context.Background(), p.config)
		if err != nil {
			p.loadErr = fmt.Errorf("run %s: %w", coveragefeature.Name, err)

			return
		}

		p.violations = make(map[string]coveragefeature.Result, run.Report.Failed)
		for _, result := range run.Report.Results {
			if result.Violation {
				p.violations[result.ImportPath] = result
			}
		}
	})

	if p.loadErr != nil {
		return nil, p.loadErr
	}

	return []*analysis.Analyzer{{
		Name:       coveragefeature.Name,
		Doc:        "enforce minimum Go test coverage",
		Run:        p.run,
		ResultType: reflect.TypeOf(struct{}{}),
	}}, nil
}

func (*plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

func (p *plugin) run(pass *analysis.Pass) (any, error) {
	if pass.Pkg == nil || pass.Pkg.Path() == "" {
		return emptyAnalyzerResult(), nil
	}

	result, ok := p.violations[pass.Pkg.Path()]
	if !ok {
		return emptyAnalyzerResult(), nil
	}

	if _, loaded := p.reported.LoadOrStore(result.ImportPath, struct{}{}); loaded {
		return emptyAnalyzerResult(), nil
	}

	pass.Report(analysis.Diagnostic{
		Pos:            diagnosticPosition(pass, result.File),
		End:            token.NoPos,
		Category:       coveragefeature.Name,
		Message:        result.Message,
		URL:            "",
		SuggestedFixes: nil,
		Related:        nil,
	})

	return emptyAnalyzerResult(), nil
}

func emptyAnalyzerResult() any {
	return nil
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
