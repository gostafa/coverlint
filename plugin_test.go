package coverlint_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
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

	plugin, err := registeredPlugin(t)(fixtureSettings(dir))
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	assertLoadMode(t, plugin)

	analyzers := buildAnalyzersForTest(t, plugin)
	assertAnalyzer(t, analyzers)
	validateAnalyzers(t, analyzers)
	assertBuildAnalyzersAgain(t, plugin)
}

func fixtureSettings(dir string) map[string]any {
	return map[string]any{
		settingMin:      100,
		settingPackages: []string{dir},
		settingTimeout:  time.Minute.String(),
		settingTestArgs: []string{runFlag, testAddName},
	}
}

func assertLoadMode(t *testing.T, plugin register.LinterPlugin) {
	t.Helper()

	if got := plugin.GetLoadMode(); got != register.LoadModeSyntax {
		t.Fatalf("GetLoadMode() = %q, want %q", got, register.LoadModeSyntax)
	}
}

func buildAnalyzersForTest(t *testing.T, plugin register.LinterPlugin) []*analysis.Analyzer {
	t.Helper()

	analyzers, err := plugin.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers: %v", err)
	}

	return analyzers
}

func assertAnalyzer(t *testing.T, analyzers []*analysis.Analyzer) {
	t.Helper()

	if len(analyzers) != 1 {
		t.Fatalf("analyzers = %#v, want one analyzer", analyzers)
	}

	if analyzers[0].Name != coveragefeature.Name {
		t.Fatalf("analyzer name = %q, want %q", analyzers[0].Name, coveragefeature.Name)
	}

	if analyzers[0].ResultType == nil {
		t.Fatal("ResultType is nil")
	}
}

func validateAnalyzers(t *testing.T, analyzers []*analysis.Analyzer) {
	t.Helper()

	err := analysis.Validate(analyzers)
	if err != nil {
		t.Fatalf("Validate analyzers: %v", err)
	}
}

func assertBuildAnalyzersAgain(t *testing.T, plugin register.LinterPlugin) {
	t.Helper()

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
	analyzers := fixtureAnalyzers(t, dir)

	unrelatedPass, unrelatedDiagnostics := analysisPassForTest(
		t,
		analyzers[0],
		"example.com/unrelated",
		filepath.Join(dir, "unrelated.go"),
	)

	runAnalyzerForTest(t, analyzers[0], unrelatedPass, "unrelated package")
	assertNoDiagnostics(t, unrelatedDiagnostics, "unrelated")

	fixtureFile := filepath.Join(dir, "calc.go")
	pass, diagnostics := analysisPassForTest(
		t,
		analyzers[0],
		fixtureImportPath(dir),
		fixtureFile,
	)

	runAnalyzerForTest(t, analyzers[0], pass, "fixture package")
	runAnalyzerForTest(t, analyzers[0], pass, "second fixture package")
	assertFixtureDiagnostic(t, pass, diagnostics, fixtureFile)
}

func fixtureAnalyzers(t *testing.T, dir string) []*analysis.Analyzer {
	t.Helper()

	plugin, err := registeredPlugin(t)(fixtureSettings(dir))
	if err != nil {
		t.Fatalf("construct plugin: %v", err)
	}

	return buildAnalyzersForTest(t, plugin)
}

func runAnalyzerForTest(
	t *testing.T,
	analyzer *analysis.Analyzer,
	pass *analysis.Pass,
	label string,
) {
	t.Helper()

	result, err := analyzer.Run(pass)
	if err != nil {
		t.Fatalf("Run %s: %v", label, err)
	}

	assertAnalyzerResultType(t, analyzer, result, label)
}

func assertAnalyzerResultType(t *testing.T, analyzer *analysis.Analyzer, result any, label string) {
	t.Helper()

	if result == nil {
		t.Fatalf("Run %s result is nil, want %v", label, analyzer.ResultType)
	}

	if got := reflect.TypeOf(result); got != analyzer.ResultType {
		t.Fatalf("Run %s result type = %v, want %v", label, got, analyzer.ResultType)
	}
}

func assertNoDiagnostics(t *testing.T, diagnostics *[]analysis.Diagnostic, label string) {
	t.Helper()

	if len(*diagnostics) != 0 {
		t.Fatalf("%s diagnostics = %#v, want none", label, *diagnostics)
	}
}

func assertFixtureDiagnostic(
	t *testing.T,
	pass *analysis.Pass,
	diagnostics *[]analysis.Diagnostic,
	fixtureFile string,
) {
	t.Helper()

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

	assertAnalyzerResultType(t, analyzers[0], result, "missing package")
}

func TestRegisteredPluginWrapsDecodeError(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	_, err := constructor("not settings")
	if err == nil || !strings.Contains(err.Error(), "decode coverlint settings") {
		t.Fatalf("error = %v, want decode wrapper", err)
	}
}

func TestRegisteredPluginRejectsUnknownSettings(t *testing.T) {
	t.Parallel()

	constructor := registeredPlugin(t)

	_, err := constructor(map[string]any{
		"minimum": 85,
	})
	if err == nil || !strings.Contains(err.Error(), `unknown coverage config field: "minimum"`) {
		t.Fatalf("error = %v, want unknown setting error", err)
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

	file, err := parser.ParseFile(fileSet, filename, "package pkg\n", parser.PackageClauseOnly)
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

	dir := moduleFixtureDir(t)

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

func moduleFixtureDir(t *testing.T) string {
	t.Helper()

	temp := t.TempDir()
	suffix := filepath.Base(filepath.Dir(temp)) + "-" + filepath.Base(temp)
	dir := filepath.Join(".", "coverlint-fixture-"+fixtureName(t)+"-"+suffix)

	err := os.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	return dir
}
