package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
)

var (
	errBoom        = errors.New("boom")
	errWriteFailed = errors.New("write failed")
)

func TestRunPrintsVersion(t *testing.T) {
	t.Parallel()

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	code := run([]string{"-version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	if strings.TrimSpace(stdout.String()) != version {
		t.Fatalf("stdout = %q, want version", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReturnsUsageExitWhenVersionWriteFails(t *testing.T) {
	t.Parallel()

	code := run([]string{"-version"}, failingWriter{}, io.Discard)
	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}
}

func TestRunPrintsHelp(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	code := run([]string{"-h"}, io.Discard, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	if !strings.Contains(stderr.String(), "Usage: coverlint [flags]") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestRunRejectsBadFlag(t *testing.T) {
	t.Parallel()

	code := run([]string{"-bad"}, io.Discard, io.Discard)
	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}
}

func TestRunCoverageRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		opts options
	}{
		{name: "override", opts: optionsForTest(80, time.Minute, stringList{"bad"}, nil, false)},
		{name: "minimum", opts: optionsForTest(101, time.Minute, nil, nil, false)},
		{name: "timeout", opts: optionsForTest(80, 0, nil, nil, false)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer

			code := runCoverage(test.opts, nil, io.Discard, &stderr)
			if code != usageExitCode {
				t.Fatalf("exit code = %d, want usage exit", code)
			}

			if !strings.Contains(stderr.String(), "coverlint:") {
				t.Fatalf("stderr = %q, want coverlint error", stderr.String())
			}
		})
	}
}

func TestRunCoveragePrintsCheckError(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	code := runCoverage(
		optionsForTest(80, time.Minute, nil, stringList{"-covermode=atomic"}, false),
		nil,
		io.Discard,
		&stderr,
	)
	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}

	if !strings.Contains(stderr.String(), "resolve coverage config") {
		t.Fatalf("stderr = %q, want config error", stderr.String())
	}
}

func TestRunCoverageSucceedsForFixture(t *testing.T) {
	t.Parallel()

	dir := writeCLIFixture(t)

	var stderr bytes.Buffer

	code := runCoverage(
		optionsForTest(90, time.Minute, nil, stringList{"-run", "TestAdd"}, false),
		[]string{dir},
		io.Discard,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	if !strings.Contains(stderr.String(), "coverlint: passed") {
		t.Fatalf("stderr = %q, want pass report", stderr.String())
	}
}

func TestReportCoveragePrintsDiagnostics(t *testing.T) {
	t.Parallel()

	runResult := coveragefeature.Run{
		Report: coveragefeature.Report{
			Results: []coveragefeature.Result{{
				ImportPath: "example.com/pkg",
				File:       "",
				Rule:       nil,
				Coverage:   50,
				Statements: 2,
				Covered:    1,
				Skipped:    false,
				Violation:  true,
				Message:    "coverage 50.00% is below minimum 90.00%",
			}},
			Checked: 1,
			Failed:  1,
			Skipped: 0,
		},
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	code := reportCoverage(&stdout, &stderr, runResult)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}

	if !strings.Contains(stdout.String(), "example.com/pkg:1:1: coverage 50.00%") {
		t.Fatalf("stdout = %q, want diagnostic", stdout.String())
	}

	if !strings.Contains(stderr.String(), "coverlint: failed with 1 issue") {
		t.Fatalf("stderr = %q, want failed summary", stderr.String())
	}
}

func TestReportCoverageHandlesWriteErrors(t *testing.T) {
	t.Parallel()

	failedRun := coveragefeature.Run{
		Report: coveragefeature.Report{
			Results: []coveragefeature.Result{{
				ImportPath: "",
				File:       "",
				Rule:       nil,
				Coverage:   0,
				Statements: 0,
				Covered:    0,
				Skipped:    false,
				Violation:  true,
				Message:    "bad",
			}},
			Checked: 0,
			Failed:  1,
			Skipped: 0,
		},
	}

	code := reportCoverage(failingWriter{}, io.Discard, failedRun)
	if code != usageExitCode {
		t.Fatalf("diagnostic write exit code = %d, want usage exit", code)
	}

	code = reportCoverage(io.Discard, failingWriter{}, passedRunForTest())
	if code != usageExitCode {
		t.Fatalf("summary write exit code = %d, want usage exit", code)
	}
}

func TestPrintUsageWritesFullUsage(t *testing.T) {
	t.Parallel()

	var (
		opts   options
		stderr bytes.Buffer
	)

	flagSet := newFlagSet(&stderr, &opts)

	err := printUsage(&stderr, flagSet)
	if err != nil {
		t.Fatalf("printUsage: %v", err)
	}

	output := stderr.String()
	for _, want := range []string{"Usage: coverlint", "Examples:", "-min"} {
		if !strings.Contains(output, want) {
			t.Fatalf("usage = %q, want %q", output, want)
		}
	}
}

func TestPrintUsageReturnsWriteError(t *testing.T) {
	t.Parallel()

	var opts options

	err := printUsage(failingWriter{}, newFlagSet(io.Discard, &opts))
	if err == nil {
		t.Fatal("printUsage succeeded, want error")
	}
}

func TestPrintUsageReturnsLaterWriteError(t *testing.T) {
	t.Parallel()

	for _, remaining := range []int{1, 2, 3, 4, 5} {
		t.Run(fmt.Sprintf("after_%d", remaining), func(t *testing.T) {
			t.Parallel()

			var opts options

			writer := &failAfterWriter{remaining: remaining}

			err := printUsage(writer, newFlagSet(io.Discard, &opts))
			if err == nil {
				t.Fatal("printUsage succeeded, want later write error")
			}
		})
	}
}

func TestOpenWebIfRequested(t *testing.T) {
	t.Parallel()

	code := openWebIfRequested(
		optionsForTest(0, 0, nil, nil, false),
		passedRunForTest(),
		io.Discard,
		io.Discard,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 when web disabled", code)
	}

	var stderr bytes.Buffer

	code = openWebIfRequested(
		optionsForTest(0, time.Minute, nil, nil, true),
		passedRunForTest(),
		io.Discard,
		&stderr,
	)
	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}
}

func TestWritersWrapErrors(t *testing.T) {
	t.Parallel()

	err := writeLine(failingWriter{}, "hello")
	if err == nil {
		t.Fatal("writeLine succeeded, want error")
	}

	err = writeFormatted(failingWriter{}, "%s", "hello")
	if err == nil {
		t.Fatal("writeFormatted succeeded, want error")
	}

	err = printError(failingWriter{}, errBoom)
	if err == nil {
		t.Fatal("printError succeeded, want error")
	}
}

func TestStringList(t *testing.T) {
	t.Parallel()

	var list stringList

	err := list.Set("one")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	err = list.Set("two")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	if got := list.String(); got != "one,two" {
		t.Fatalf("String() = %q, want joined values", got)
	}
}

func TestParseOverridesUsesGlobPattern(t *testing.T) {
	t.Parallel()

	overrides, err := parseOverrides([]string{"**/internal/**=85", "**/critical/**=95%"})
	if err != nil {
		t.Fatalf("parseOverrides: %v", err)
	}

	if len(overrides) != 2 {
		t.Fatalf("len(overrides) = %d, want 2", len(overrides))
	}

	assertOverride(t, overrides[0], "**/internal/**", 85)
	assertOverride(t, overrides[1], "**/critical/**", 95)
}

func assertOverride(
	t *testing.T,
	override coveragefeature.Override,
	pattern string,
	minimum float64,
) {
	t.Helper()

	if override.Pattern != pattern {
		t.Fatalf("override pattern = %q, want %q", override.Pattern, pattern)
	}

	if override.Min != minimum {
		t.Fatalf("override minimum = %v, want %v", override.Min, minimum)
	}
}

func TestParseOverridesErrorUsesGlobTerminology(t *testing.T) {
	t.Parallel()

	_, err := parseOverrides([]string{"invalid"})
	if err == nil {
		t.Fatal("parseOverrides succeeded, want error")
	}

	want := `override 1 "invalid" must have the form GLOB=MIN`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}

	if !errors.Is(err, errOverrideFormat) {
		t.Fatalf("errors.Is(%v, errOverrideFormat) = false", err)
	}
}

func TestParseOverridesRejectsInvalidMinimum(t *testing.T) {
	t.Parallel()

	_, err := parseOverrides([]string{"**=nope"})
	if err == nil || !strings.Contains(err.Error(), "invalid minimum") {
		t.Fatalf("error = %v, want invalid minimum", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errWriteFailed
}

type failAfterWriter struct {
	remaining int
}

func (w *failAfterWriter) Write(data []byte) (int, error) {
	if w.remaining == 0 {
		return 0, errWriteFailed
	}

	w.remaining--

	return len(data), nil
}

func optionsForTest(
	minimum float64,
	timeout time.Duration,
	overrides stringList,
	testArgs stringList,
	web bool,
) options {
	return options{
		min:         minimum,
		overrides:   overrides,
		excludes:    nil,
		timeout:     timeout,
		testArgs:    testArgs,
		web:         web,
		showVersion: false,
	}
}

func passedRunForTest() coveragefeature.Run {
	return coveragefeature.Run{
		Report: coveragefeature.Report{
			Results: nil,
			Checked: 0,
			Failed:  0,
			Skipped: 0,
		},
	}
}

func writeCLIFixture(t *testing.T) string {
	t.Helper()

	dir := moduleFixtureDir(t)

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	writeCLIFile(
		t,
		dir,
		"calc.go",
		"package clifixture\n\nfunc Add(a, b int) int { return a + b }\n",
	)
	writeCLIFile(t, dir, "calc_test.go", `package clifixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	return "./" + filepath.Base(dir)
}

func writeCLIFile(t *testing.T, dir, name, content string) {
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
