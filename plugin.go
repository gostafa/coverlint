// Package coverlint registers the golangci-lint coverlint plugin.
package coverlint

import (
	"context"
	"fmt"
	"go/token"
	"sync"

	"github.com/golangci/plugin-module-register/register"
	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
	"golang.org/x/tools/go/analysis"
)

var _ = registerPlugin()

func registerPlugin() bool {
	register.Plugin(
		coveragefeature.Name,
		func(rawSettings any) (register.LinterPlugin, error) {
			return newPlugin(rawSettings)
		},
	)

	return true
}

type plugin struct {
	config coveragefeature.Config

	loadOnce   sync.Once
	loadErr    error
	violations []coveragefeature.Result
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
		violations: nil,
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

		p.violations = make([]coveragefeature.Result, 0, run.Report.Failed)
		for _, result := range run.Report.Results {
			if result.Violation {
				p.violations = append(p.violations, result)
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

	position := token.NoPos
	if len(pass.Files) > 0 {
		position = pass.Files[0].Package
	}

	for _, result := range p.violations {
		if _, loaded := p.reported.LoadOrStore(result.ImportPath, struct{}{}); loaded {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Pos:            position,
			End:            token.NoPos,
			Category:       coveragefeature.Name,
			Message:        result.Message,
			URL:            "",
			SuggestedFixes: nil,
			Related:        nil,
		})
	}

	return nil, nil
}
