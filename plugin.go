// Package coverlint registers the golangci-lint coverlint plugin.
package coverlint

import (
	"context"
	"fmt"
	"go/token"
	"path/filepath"
	"sync"
	"github.com/golangci/plugin-module-register/register"
	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin(
		coveragefeature.Name,
		func(rawSettings any) (register.LinterPlugin, error) {
			return newPlugin(rawSettings)
		},
	)
}

type plugin struct {
	config coveragefeature.Config
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
		Name: coveragefeature.Name,
		Doc:  "enforce minimum Go test coverage",
		Run:  p.run,
	}}, nil
}

func (*plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

func (p *plugin) run(pass *analysis.Pass) (any, error) {
	if pass.Pkg == nil || pass.Pkg.Path() == "" {
		return nil, nil
	}

	result, ok := p.violations[pass.Pkg.Path()]
	if !ok {
		return nil, nil
	}

	if _, loaded := p.reported.LoadOrStore(result.ImportPath, struct{}{}); loaded {
		return nil, nil
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

	return nil, nil
}

func diagnosticPosition(pass *analysis.Pass, resultFile string) token.Pos {
	if pass.Fset != nil && resultFile != "" {
		wanted := normalizeFilename(resultFile)

		for _, file := range pass.Files {
			if file == nil {
				continue
			}

			position := pass.Fset.PositionFor(file.Package, true)
			if normalizeFilename(position.Filename) == wanted {
				return file.Package
			}
		}
	}

	for _, file := range pass.Files {
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
