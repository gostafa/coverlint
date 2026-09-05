// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gostafa/coverlint/coverlint"
	coveragedomain "github.com/gostafa/coverlint/internal/features/coverage/domain"
	reportingdomain "github.com/gostafa/coverlint/internal/features/reporting/domain"
)

func TestCheckRunsCoverage(t *testing.T) {
	t.Parallel()

	dir := writeCoverageFixture(t)

	run, err := coverlint.Check(
		t.Context(),
		configForTest(0.90, time.Minute.String()),
		"./"+filepath.Base(dir),
	)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if run.Report.Checked != 1 || run.Report.Failed != 0 {
		t.Fatalf("Report = %#v", run.Report)
	}
}

func TestCheckWritesResultFiles(t *testing.T) {
	t.Parallel()

	dir := writeCoverageFixture(t)
	outDir := t.TempDir()
	testPath := filepath.Join(outDir, "subdir", "test.txt")
	coveragePath := filepath.Join(outDir, "subdir", "coverage.out")

	cfg := configForTest(0, time.Minute.String())
	cfg.TestResultPath = testPath
	cfg.CoverageResultPath = coveragePath
	cfg.TestArgs = []string{"-run", "TestAdd"}

	_, err := coverlint.Check(t.Context(), cfg, "./"+filepath.Base(dir))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	testData, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile test result: %v", err)
	}

	if len(bytes.TrimSpace(testData)) == 0 {
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

func TestCheckWritesRelativeResultPaths(t *testing.T) {
	t.Parallel()

	dir := writeCoverageFixture(t)
	outDir := t.TempDir()

	testPath, coveragePath := relativeResultPaths(t, outDir, "nested/test.txt", "coverage.out")

	cfg := configForTest(0, time.Minute.String())
	cfg.TestResultPath = testPath
	cfg.CoverageResultPath = coveragePath
	cfg.TestArgs = []string{"-run", "TestAdd"}

	_, err := coverlint.Check(t.Context(), cfg, "./"+filepath.Base(dir))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	testData, err := os.ReadFile(filepath.Join(outDir, "nested", "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile test result: %v", err)
	}

	if len(bytes.TrimSpace(testData)) == 0 {
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

func TestCheckSkipsResultFilesWhenPathsEmpty(t *testing.T) {
	t.Parallel()

	dir := writeCoverageFixture(t)
	outDir := t.TempDir()

	cfg := configForTest(0, time.Minute.String())
	cfg.TestArgs = []string{"-run", "TestAdd"}

	_, err := coverlint.Check(t.Context(), cfg, "./"+filepath.Base(dir))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("unexpected files written: %#v", entries)
	}
}

func TestCheckWrapsResultFileWriteErrors(t *testing.T) {
	t.Parallel()

	dir := writeCoverageFixture(t)
	outDir := t.TempDir()
	blockPath := filepath.Join(outDir, "block")

	err := os.WriteFile(blockPath, []byte("not-a-dir"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile block: %v", err)
	}

	cfg := configForTest(0, time.Minute.String())
	cfg.TestArgs = []string{"-run", "TestAdd"}
	cfg.TestResultPath = filepath.Join(blockPath, "out.txt")

	_, err = coverlint.Check(t.Context(), cfg, "./"+filepath.Base(dir))

	if err == nil || !strings.Contains(err.Error(), "Check:") {
		t.Fatalf("error = %v, want Check wrapper for test result write", err)
	}

	if !strings.Contains(err.Error(), "write test result file:") {
		t.Fatalf("error = %v, want write test result file wrap", err)
	}

	cfg.TestResultPath = ""
	cfg.CoverageResultPath = filepath.Join(blockPath, "coverage.out")

	_, err = coverlint.Check(t.Context(), cfg, "./"+filepath.Base(dir))

	if err == nil || !strings.Contains(err.Error(), "Check:") {
		t.Fatalf("error = %v, want Check wrapper for coverage write", err)
	}

	if !strings.Contains(err.Error(), "write coverage result file:") {
		t.Fatalf("error = %v, want write coverage result file wrap", err)
	}

	cfg.CoverageResultPath = t.TempDir()

	_, err = coverlint.Check(t.Context(), cfg, "./"+filepath.Base(dir))

	if err == nil || !strings.Contains(err.Error(), "write result file") {
		t.Fatalf("error = %v, want write result file failure for directory path", err)
	}
}

func writeCoverageFixture(t *testing.T) string {
	t.Helper()

	dir := moduleFixtureDir(t)

	t.Cleanup(func() { removeCoverageFixture(t, dir) })

	writeFile(t, dir, "calc.go", "package fixture\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile(t, dir, "calc_test.go", `package fixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	return dir
}

func removeCoverageFixture(t *testing.T, dir string) {
	t.Helper()

	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("RemoveAll %s: %v", dir, err)
	}
}

func TestCheckWrapsConfigErrors(t *testing.T) {
	t.Parallel()

	_, err := coverlint.Check(t.Context(), configForTest(1.01, ""))

	if err == nil || !strings.Contains(err.Error(), "resolve coverage config") {
		t.Fatalf("error = %v, want config wrapper", err)
	}
}

func TestValidateMinimumWrapsDomainError(t *testing.T) {
	t.Parallel()

	err := coverlint.ValidateMinimum(1.01)

	if err == nil || !strings.Contains(err.Error(), "validate coverage minimum") {
		t.Fatalf("error = %v, want validate wrapper", err)
	}
}

func TestValidateMinimumAcceptsValidValue(t *testing.T) {
	t.Parallel()

	err := coverlint.ValidateMinimum(0.90)
	if err != nil {
		t.Fatalf("ValidateMinimum: %v", err)
	}
}

func TestRunOpenWebRequiresReporter(t *testing.T) {
	t.Parallel()

	err := (&coverlint.Run{Report: coverlint.Report{Results: nil, Checked: 0, Failed: 0, Skipped: 0}}).OpenWeb(
		t.Context(),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
	)

	if err == nil || !strings.Contains(err.Error(), "HTML coverage adapter is not configured") {
		t.Fatalf("error = %v, want missing reporter", err)
	}
}

func TestDiagnosticDelegatesToReportingDomain(t *testing.T) {
	t.Parallel()

	got := reportingdomain.Diagnostic(&coveragedomain.Result{
		ImportPath: "example.com/pkg",
		File:       "",
		Rule:       nil,
		Coverage:   0,
		Statements: 0,
		Covered:    0,
		Skipped:    false,
		Violation:  false,
		Message:    "coverage below 90%",
	}, coverlint.Name)

	want := "example.com/pkg:1:1: coverage below 90% (coverlint)"

	if got != want {
		t.Fatalf("Diagnostic() = %q, want %q", got, want)
	}
}

func configForTest(minimum float64, timeout string) *coverlint.Config {
	return &coverlint.Config{
		Rules:    []coveragedomain.Rule{{Pattern: "**", Min: minimum}},
		Packages: nil,
		Timeout:  timeout,
		TestArgs: nil,
	}
}

func writeFile(t *testing.T, dir, name, content string) {
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
