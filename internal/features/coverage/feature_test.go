package coverage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coverage "github.com/gostafa/coverlint/internal/features/coverage"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func TestCheckRunsCoverage(t *testing.T) {
	t.Parallel()

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

	writeFile(t, dir, "calc.go", "package fixture\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile(t, dir, "calc_test.go", `package fixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	run, err := coverage.Check(
		context.Background(),
		configForTest(90, time.Minute.String()),
		"./"+filepath.Base(dir),
	)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if run.Report.Checked != 1 || run.Report.Failed != 0 {
		t.Fatalf("Report = %#v", run.Report)
	}
}

func TestCheckWrapsConfigErrors(t *testing.T) {
	t.Parallel()

	_, err := coverage.Check(context.Background(), configForTest(101, ""))
	if err == nil || !strings.Contains(err.Error(), "resolve coverage config") {
		t.Fatalf("error = %v, want config wrapper", err)
	}
}

func TestValidateMinimumWrapsDomainError(t *testing.T) {
	t.Parallel()

	err := coverage.ValidateMinimum(101)
	if err == nil || !strings.Contains(err.Error(), "validate coverage minimum") {
		t.Fatalf("error = %v, want validate wrapper", err)
	}
}

func TestValidateMinimumAcceptsValidValue(t *testing.T) {
	t.Parallel()

	err := coverage.ValidateMinimum(90)
	if err != nil {
		t.Fatalf("ValidateMinimum: %v", err)
	}
}

func TestRunOpenWebRequiresReporter(t *testing.T) {
	t.Parallel()

	err := (coverage.Run{Report: coverage.Report{Results: nil, Checked: 0, Failed: 0, Skipped: 0}}).OpenWeb(
		context.Background(),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
	)
	if err == nil || !strings.Contains(err.Error(), "HTML coverage adapter is not configured") {
		t.Fatalf("error = %v, want missing reporter", err)
	}
}

func TestDiagnosticDelegatesToTextAdapter(t *testing.T) {
	t.Parallel()

	got := coverage.Diagnostic(domain.Result{
		ImportPath: "example.com/pkg",
		File:       "",
		Rule:       nil,
		Coverage:   0,
		Statements: 0,
		Covered:    0,
		Skipped:    false,
		Violation:  false,
		Message:    "coverage below 90%",
	})

	want := "example.com/pkg:1:1: coverage below 90% (coverlint)"
	if got != want {
		t.Fatalf("Diagnostic() = %q, want %q", got, want)
	}
}

func configForTest(minimum float64, timeout string) coverage.Config {
	return coverage.Config{
		Min:       minimum,
		Overrides: nil,
		Exclude:   nil,
		Packages:  nil,
		Timeout:   timeout,
		TestArgs:  nil,
	}
}

func writeFile(t *testing.T, dir, name, content string) {
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
