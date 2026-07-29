package coverlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestDiagnosticPositionFallsBackToFirstAvailableFile(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(
		fileSet,
		"fallback.go",
		"package fixture\n",
		parser.PackageClauseOnly,
	)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	pass := new(analysis.Pass)
	pass.Fset = fileSet
	pass.Files = []*ast.File{nil, file}

	if got := diagnosticPosition(pass, "missing.go"); got != file.Package {
		t.Fatalf("diagnosticPosition() = %v, want %v", got, file.Package)
	}
}
