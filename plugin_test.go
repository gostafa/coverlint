package coverlint

import (
	"go/types"
	"testing"

	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
	"golang.org/x/tools/go/analysis"
)

func TestPluginReportsAllViolationsFromFirstAnalyzedPackage(t *testing.T) {
	t.Parallel()

	p := &plugin{violations: []coveragefeature.Result{
		{ImportPath: "github.com/acme/project/a", Message: "a failed"},
		{ImportPath: "github.com/acme/project/b", Message: "b failed"},
	}}
	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Pkg: types.NewPackage("github.com/acme/project/loaded", "loaded"),
		Report: func(diagnostic analysis.Diagnostic) {
			diagnostics = append(diagnostics, diagnostic)
		},
	}

	if _, err := p.run(pass); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(diagnostics) != 2 {
		t.Fatalf("len(diagnostics) = %d, want 2", len(diagnostics))
	}
	if diagnostics[0].Message != "a failed" || diagnostics[1].Message != "b failed" {
		t.Fatalf("diagnostics = %#v, want both violation messages", diagnostics)
	}

	diagnostics = nil
	if _, err := p.run(pass); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("second run reported %#v, want no duplicate diagnostics", diagnostics)
	}
}
