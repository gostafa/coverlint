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
	settingTestArgs = "test-args"
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


	if analyzers[0].ResultType != nil {
		t.Fatalf("ResultType = %v, want nil", analyzers[0].ResultType)
	}

	if err := analysis.Validate(analyzers); err != nil {
		t.Fatalf("Validate analyzers: %v", err)
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

	unrelatedPass, unrelatedDiagnostics := analysisPassForTest(
		t,
		analyzers[0],
		"example.com/unrelated",
		filepath.Join(dir, "unrelated.go"),
	)

	result, err := analyzers[0].Run(unrelatedPass)
	if err != nil {
		t.Fatalf("Run unrelated package: %v", err)
	}
	if result != nil {
		t.Fatalf("unrelated Run result = %#v, want nil", result)
	}
	if len(*unrelatedDiagnostics) != 0 {
		t.Fatalf("unrelated diagnostics = %#v, want none", *unrelatedDiagnostics)
	}

	fixtureFile := filepath.Join(dir, "calc.go")
	pass, diagnostics := analysisPassForTest(
		t,
		analyzers[0],
		fixtureImportPath(dir),
		fixtureFile,
	)

	result, err = analyzers[0].Run(pass)
	if err != nil {
		t.Fatalf("Run fixture package: %v", err)
	}
	if result != nil {
		t.Fatalf("Run result = %#v, want nil for nil ResultType", result)
	}

	result, err = analyzers[0].Run(pass)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if result != nil {
		t.Fatalf("second Run result = %#v, want nil for nil ResultType", result)
	}

	if len(*diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one unique report", *diagnostics)
	}

	diagnostic := (*diagnostics)[0]
	if diagnostic.Category != coveragefeature.Name {
		t.Fatalf("category = %q, want %q", diagnostic.Category, coveragefeature.Name)
	}

	gotFile := pass.Fset.PositionFor(diagnostic.Pos, true).Filename
	if !sameFilename(gotFile, fixtureFile) {
		t.Fatalf("diagnostic file = %q, want %q", gotFile, fixtureFile)
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


	result, err := analyzers[0].Run(pass)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != nil {
		t.Fatalf("Run result = %#v, want nil for nil ResultType", result)
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
	packagePath string,
	filename string,
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
		Pkg:          types.NewPackage(packagePath, "pluginfixture"),
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

func fixtureImportPath(dir string) string {
	return "github.com/gostafa/coverlint/" + filepath.ToSlash(filepath.Base(dir))
}

func sameFilename(left, right string) bool {
	leftAbsolute, leftErr := filepath.Abs(left)
	rightAbsolute, rightErr := filepath.Abs(right)

	if leftErr == nil {
		left = leftAbsolute
	}
	if rightErr == nil {
		right = rightAbsolute
	}

	return filepath.Clean(left) == filepath.Clean(right)
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
