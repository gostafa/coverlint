package coverlint_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golangci/plugin-module-register/register"
	_ "github.com/gostafa/coverlint"
	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
	"golang.org/x/tools/go/analysis"
)

const (
	settingMin      = "min"
	settingPackages = "packages"
	settingTestArgs = "testArgs"
	settingTimeout  = "timeout"
	runFlag         = "-run"
	testAddName     = "TestAdd"
)

func TestRegisteredPluginBuildsAnalyzer(t *testing.T) {
	t.Parallel()

	dir := writePluginFixture(t)
	constructor := registeredPlugin(t)

	plugin, err := constructor(map[string]any{
		settingMin: 100,
		settingPackages: []string{
			dir,
		},
		settingTimeout: time.Minute.String(),
		settingTestArgs: []string{
			runFlag, testAddName,
		},
	})
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	if got := plugin.GetLoadMode(); got != register.LoadModeSyntax {
		t.Fatalf("GetLoadMode() = %q, want %q", got, register.LoadModeSyntax)
	}

	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}

	if len(analyzers) != 1 || analyzers[0].Name != coveragefeature.Name {
		t.Fatalf("analyzers = %#v", analyzers)
	}

	again, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("second BuildAnalyzers: %v", err)
	}

	if again[0].Run == nil {
		t.Fatal("analyzer Run is nil")
	}
}

func TestRegisteredPluginBuildAnalyzersReturnsLoadError(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	plugin, err := constructor(map[string]any{
		settingMin: 101,
	})
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	_, err = plugin.BuildAnalyzers()
	if err == nil || !strings.Contains(err.Error(), "run coverlint") {
		t.Fatalf("error = %v, want load wrapper", err)
	}
}

func TestRegisteredPluginRunReportsViolationsOnce(t *testing.T) {
	t.Parallel()

	dir := writePluginFixture(t)
	constructor := registeredPlugin(t)

	plugin, err := constructor(map[string]any{
		settingMin:      100,
		settingPackages: []string{dir},
		settingTimeout:  time.Minute.String(),
		settingTestArgs: []string{runFlag, testAddName},
	})
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}

	pass, diagnostics := analysisPassForTest(t, analyzers[0])

	_, err = analyzers[0].Run(pass)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	_, err = analyzers[0].Run(pass)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if len(*diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one unique report", *diagnostics)
	}

	if (*diagnostics)[0].Category != coveragefeature.Name {
		t.Fatalf("category = %q, want %q", (*diagnostics)[0].Category, coveragefeature.Name)
	}
}

func TestRegisteredPluginIgnoresMissingPackage(t *testing.T) {
	t.Parallel()

	dir := writePluginFixture(t)
	constructor := registeredPlugin(t)

	plugin, err := constructor(map[string]any{
		settingMin:      100,
		settingPackages: []string{dir},
		settingTimeout:  time.Minute.String(),
		settingTestArgs: []string{runFlag, testAddName},
	})
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}

	pass := passWithMissingPackageForTest(analyzers[0])

	_, err = analyzers[0].Run(pass)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRegisteredPluginWrapsDecodeError(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	_, err := constructor("not settings")
	if err == nil || !strings.Contains(err.Error(), "decode coverlint settings") {
		t.Fatalf("error = %v, want decode wrapper", err)
	}
}

func analysisPassForTest(
	t *testing.T,
	analyzer *analysis.Analyzer,
) (*analysis.Pass, *[]analysis.Diagnostic) {
	t.Helper()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "pkg.go", "package pkg\n", parser.PackageClauseOnly)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	diagnostics := make([]analysis.Diagnostic, 0, 1)

	return &analysis.Pass{
		Analyzer:     analyzer,
		Fset:         fileSet,
		Files:        []*ast.File{file},
		OtherFiles:   nil,
		IgnoredFiles: nil,
		Pkg:          types.NewPackage("example.com/pkg", "pkg"),
		TypesInfo:    nil,
		TypesSizes:   nil,
		TypeErrors:   nil,
		Module:       nil,
		Report: func(diagnostic analysis.Diagnostic) {
			diagnostics = append(diagnostics, diagnostic)
		},
		ResultOf:          nil,
		ReadFile:          nil,
		ImportObjectFact:  nil,
		ImportPackageFact: nil,
		ExportObjectFact:  nil,
		ExportPackageFact: nil,
		AllPackageFacts:   nil,
		AllObjectFacts:    nil,
	}, &diagnostics
}

func passWithMissingPackageForTest(analyzer *analysis.Analyzer) *analysis.Pass {
	return &analysis.Pass{
		Analyzer:          analyzer,
		Fset:              nil,
		Files:             nil,
		OtherFiles:        nil,
		IgnoredFiles:      nil,
		Pkg:               nil,
		TypesInfo:         nil,
		TypesSizes:        nil,
		TypeErrors:        nil,
		Module:            nil,
		Report:            func(analysis.Diagnostic) {},
		ResultOf:          nil,
		ReadFile:          nil,
		ImportObjectFact:  nil,
		ImportPackageFact: nil,
		ExportObjectFact:  nil,
		ExportPackageFact: nil,
		AllPackageFacts:   nil,
		AllObjectFacts:    nil,
	}
}

func registeredPlugin(t *testing.T) register.NewPlugin {
	t.Helper()

	constructor, err := register.GetPlugin(coveragefeature.Name)
	if err != nil {
		t.Fatalf("GetPlugin: %v", err)
	}

	return constructor
}

func writePluginFixture(t *testing.T) string {
	t.Helper()

	dir := filepath.Join(".", "coverlint-fixture-"+fixtureName(t))

	err := os.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	writePluginFile(
		t,
		dir,
		"calc.go",
		"package pluginfixture\n\nfunc Add(a, b int) int { return a + b }\nfunc Missing() int { return 1 }\n",
	)
	writePluginFile(t, dir, "calc_test.go", `package pluginfixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	return "./" + filepath.Base(dir)
}

func writePluginFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	if err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

func fixtureName(t *testing.T) string {
	t.Helper()

	return strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
}
