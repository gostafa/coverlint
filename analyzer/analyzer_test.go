// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer_test

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

	"github.com/gostafa/coverlint/analyzer"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"golang.org/x/tools/go/analysis"
)

const (
	runFlag     = "-run"
	testAddName = "TestAdd"
)

func TestAnalyzerRunReportsViolationsOnce(t *testing.T) {
	t.Parallel()

	dir := writeAnalyzerFixture(t)
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

func TestAnalyzerReportsTestFailures(t *testing.T) {
	t.Parallel()

	dir := writeAnalyzerFailingFixture(t)

	a, err := analyzer.New(&analyzer.Settings{
		Rules:    []domain.Rule{{Pattern: "**", Min: 0}},
		Packages: []string{dir},
		Timeout:  time.Minute.String(),
		TestArgs: []string{runFlag, "TestFail"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	fixtureFile := filepath.Join(dir, "calc.go")
	pass, diagnostics := analysisPassForTest(
		t,
		a,
		fixtureImportPath(dir),
		fixtureFile,
	)

	runAnalyzerForTest(t, a, pass, "failing tests")

	if len(*diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one test-failure report", *diagnostics)
	}

	if !strings.Contains((*diagnostics)[0].Message, "tests failed") {
		t.Fatalf("diagnostic = %#v, want tests failed", (*diagnostics)[0])
	}
}

func TestAnalyzerWritesResultPaths(t *testing.T) {
	t.Parallel()

	dir := writeAnalyzerFixture(t)
	outDir := t.TempDir()
	testPath := filepath.Join(outDir, "plugin-test.txt")
	coveragePath := filepath.Join(outDir, "plugin.coverprofile")

	settings := fixtureAnalyzerSettings(dir)
	settings.Rules = []domain.Rule{{Pattern: "**", Min: 0}}
	settings.TestResultPath = testPath
	settings.CoverageResultPath = coveragePath

	_, err := analyzer.New(settings)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	testData, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile test result: %v", err)
	}

	if len(strings.TrimSpace(string(testData))) == 0 {
		t.Fatal("test result file is empty")
	}

	coverageData, err := os.ReadFile(coveragePath)
	if err != nil {
		t.Fatalf("ReadFile coverage result: %v", err)
	}

	if !strings.HasPrefix(string(coverageData), "mode: atomic\n") {
		t.Fatalf("coverage result = %q, want coverprofile header", coverageData)
	}
}

func TestAnalyzerWritesRelativeResultPaths(t *testing.T) {
	t.Parallel()

	dir := writeAnalyzerFixture(t)
	outDir := t.TempDir()
	testPath, coveragePath := relativeResultPaths(t, outDir, "plugin-test.txt", "coverage.out")

	settings := fixtureAnalyzerSettings(dir)
	settings.Rules = []domain.Rule{{Pattern: "**", Min: 0}}
	settings.TestResultPath = testPath
	settings.CoverageResultPath = coveragePath

	_, err := analyzer.New(settings)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	testData, err := os.ReadFile(filepath.Join(outDir, "plugin-test.txt"))
	if err != nil {
		t.Fatalf("ReadFile test result: %v", err)
	}

	if len(strings.TrimSpace(string(testData))) == 0 {
		t.Fatal("test result file is empty")
	}

	coverageData, err := os.ReadFile(filepath.Join(outDir, "coverage.out"))
	if err != nil {
		t.Fatalf("ReadFile coverage result: %v", err)
	}

	if !strings.HasPrefix(string(coverageData), "mode: atomic\n") {
		t.Fatalf("coverage result = %q, want coverprofile header", coverageData)
	}
}

func TestAnalyzerIgnoresMissingPackage(t *testing.T) {
	t.Parallel()

	dir := writeAnalyzerFixture(t)

	a, err := analyzer.New(fixtureAnalyzerSettings(dir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	pass := passWithMissingPackageForTest(a)

	result, err := a.Run(pass)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	assertAnalyzerResultType(t, a, result, "missing package")
}

func TestDiagnosticPositionFallsBackToFirstAvailableFile(t *testing.T) {
	t.Parallel()

	dir := writeAnalyzerFixture(t)

	a, err := analyzer.New(fixtureAnalyzerSettings(dir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	fileSet := token.NewFileSet()

	file, parseErr := parser.ParseFile(
		fileSet,
		"fallback.go",
		"package fixture\n",
		parser.PackageClauseOnly,
	)
	if parseErr != nil {
		t.Fatalf("ParseFile: %v", parseErr)
	}

	pass := &analysis.Pass{
		Analyzer:          a,
		Fset:              fileSet,
		Files:             []*ast.File{nil, file},
		OtherFiles:        nil,
		IgnoredFiles:      nil,
		Pkg:               types.NewPackage("example.com/nonexistent", "fixture"),
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

	result, err := a.Run(pass)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestUnmarshalSettingsAcceptsKebabTestArgs(t *testing.T) {
	t.Parallel()

	var settings analyzer.Settings

	err := analyzer.UnmarshalSettings([]byte(`{"test-args":["-race"]}`), &settings)
	if err != nil {
		t.Fatalf("UnmarshalSettings: %v", err)
	}

	if len(settings.TestArgs) != 1 || settings.TestArgs[0] != "-race" {
		t.Fatalf("TestArgs = %#v", settings.TestArgs)
	}
}

func TestUnmarshalSettingsAcceptsKebabResultPaths(t *testing.T) {
	t.Parallel()

	var settings analyzer.Settings

	err := analyzer.UnmarshalSettings(
		[]byte(`{"test-result-path":"/tmp/test.txt","coverage-result-path":"/tmp/c.out"}`),
		&settings,
	)
	if err != nil {
		t.Fatalf("UnmarshalSettings: %v", err)
	}

	if settings.TestResultPath != "/tmp/test.txt" {
		t.Fatalf("TestResultPath = %q", settings.TestResultPath)
	}

	if settings.CoverageResultPath != "/tmp/c.out" {
		t.Fatalf("CoverageResultPath = %q", settings.CoverageResultPath)
	}
}

func TestUnmarshalSettingsRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	var settings analyzer.Settings

	err := analyzer.UnmarshalSettings([]byte(`{"minimum":85}`), &settings)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func fixtureAnalyzers(t *testing.T, dir string) []*analysis.Analyzer {
	t.Helper()

	a, err := analyzer.New(fixtureAnalyzerSettings(dir))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return []*analysis.Analyzer{a}
}

func fixtureAnalyzerSettings(dir string) *analyzer.Settings {
	return &analyzer.Settings{
		Rules:    []domain.Rule{{Pattern: "**", Min: 1}},
		Packages: []string{dir},
		Timeout:  time.Minute.String(),
		TestArgs: []string{runFlag, testAddName},
	}
}

func runAnalyzerForTest(
	t *testing.T,
	a *analysis.Analyzer,
	pass *analysis.Pass,
	label string,
) {
	t.Helper()

	result, err := a.Run(pass)
	if err != nil {
		t.Fatalf("Run %s: %v", label, err)
	}

	assertAnalyzerResultType(t, a, result, label)
}

func assertAnalyzerResultType(t *testing.T, a *analysis.Analyzer, result any, label string) {
	t.Helper()

	if result == nil {
		t.Fatalf("Run %s result is nil, want %v", label, a.ResultType)
	}

	if got := reflect.TypeOf(result); got != a.ResultType {
		t.Fatalf("Run %s result type = %v, want %v", label, got, a.ResultType)
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

	if diagnostic.Category != analyzer.Name {
		t.Fatalf("category = %q, want %q", diagnostic.Category, analyzer.Name)
	}

	gotFile := pass.Fset.PositionFor(diagnostic.Pos, true).Filename

	if !sameFilename(gotFile, fixtureFile) {
		t.Fatalf("diagnostic file = %q, want %q", gotFile, fixtureFile)
	}
}

func analysisPassForTest(
	t *testing.T,
	a *analysis.Analyzer,
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
		Analyzer:     a,
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
	return "github.com/gostafa/coverlint/analyzer/" + filepath.ToSlash(filepath.Base(dir))
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

func passWithMissingPackageForTest(a *analysis.Analyzer) *analysis.Pass {
	return &analysis.Pass{
		Analyzer:          a,
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

func writeAnalyzerFixture(t *testing.T) string {
	t.Helper()

	dir := moduleFixtureDir(t)

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	writeFixtureFile(
		t,
		dir,
		"calc.go",
		"package pluginfixture\n\nfunc Add(a, b int) int { return a + b }\nfunc Missing() int { return 1 }\n",
	)
	writeFixtureFile(t, dir, "calc_test.go", `package pluginfixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	return "./" + filepath.Base(dir)
}

func writeAnalyzerFailingFixture(t *testing.T) string {
	t.Helper()

	dir := moduleFixtureDir(t)

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	writeFixtureFile(
		t,
		dir,
		"calc.go",
		"package pluginfixture\n\nfunc Add(a, b int) int { return a + b }\n",
	)
	writeFixtureFile(t, dir, "calc_test.go", `package pluginfixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}

func TestFail(t *testing.T) {
	t.Fatal("intentional failure")
}
`)

	return "./" + filepath.Base(dir)
}

func writeFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	if err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

func relativeResultPaths(t *testing.T, outDir, testName, coverageName string) (string, string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	testPath, err := filepath.Rel(cwd, filepath.Join(outDir, filepath.FromSlash(testName)))
	if err != nil {
		t.Fatalf("Rel test path: %v", err)
	}

	coveragePath, err := filepath.Rel(cwd, filepath.Join(outDir, filepath.FromSlash(coverageName)))
	if err != nil {
		t.Fatalf("Rel coverage path: %v", err)
	}

	if filepath.IsAbs(testPath) || filepath.IsAbs(coveragePath) {
		t.Fatalf("relative paths = %q, %q, want relative", testPath, coveragePath)
	}

	return testPath, coveragePath
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
