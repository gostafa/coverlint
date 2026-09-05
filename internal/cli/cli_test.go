// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

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

	"github.com/gostafa/coverlint/coverlint"
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

	code := runCLI([]string{"-version"}, &ioStreams{stdout: &stdout, stderr: &stderr})

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

	code := runCLI([]string{"-version"}, &ioStreams{stdout: failingWriter{}, stderr: io.Discard})

	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}
}

func TestRunPrintsHelp(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	code := runCLI([]string{"-h"}, &ioStreams{stdout: io.Discard, stderr: &stderr})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	if !strings.Contains(stderr.String(), "Usage: coverlint [flags]") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestRunRejectsBadFlag(t *testing.T) {
	t.Parallel()

	code := runCLI([]string{"-bad"}, &ioStreams{stdout: io.Discard, stderr: io.Discard})

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
		{name: ruleFlag, opts: optionsForTest(time.Minute, stringList{"bad"}, nil, false)},
		{name: "minimum", opts: optionsForTest(time.Minute, stringList{"**:1.01"}, nil, false)},
		{name: timeoutFlag, opts: optionsForTest(0, nil, nil, false)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer

			code := runCoverage(&test.opts, nil, &ioStreams{stdout: io.Discard, stderr: &stderr})

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
		ptrOptions(optionsForTest(time.Minute, nil, stringList{"-covermode=atomic"}, false)),
		nil,
		&ioStreams{stdout: io.Discard, stderr: &stderr},
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
		ptrOptions(
			optionsForTest(
				time.Minute,
				stringList{"**:0.90"},
				stringList{"-run", "TestAdd"},
				false,
			),
		),
		[]string{dir},
		&ioStreams{stdout: io.Discard, stderr: &stderr},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	if !strings.Contains(stderr.String(), "coverlint: passed") {
		t.Fatalf("stderr = %q, want pass report", stderr.String())
	}
}

func TestRunCoverageWithoutResultPathsWritesNoFiles(t *testing.T) {
	t.Parallel()

	dir := writeCLIFixture(t)
	outDir := t.TempDir()

	code := runCoverage(
		ptrOptions(
			optionsForTest(
				time.Minute,
				stringList{"**:0"},
				stringList{"-run", "TestAdd"},
				false,
			),
		),
		[]string{dir},
		&ioStreams{stdout: io.Discard, stderr: io.Discard},
	)

	if code != successExitCode {
		t.Fatalf("exit code = %d, want 0", code)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("unexpected files written: %#v", entries)
	}
}

func TestRunCoverageWritesResultPaths(t *testing.T) {
	t.Parallel()

	dir := writeCLIFixture(t)
	outDir := t.TempDir()
	testPath := filepath.Join(outDir, "nested", "test.txt")
	coveragePath := filepath.Join(outDir, "nested", "coverage.out")

	opts := optionsForTest(
		time.Minute,
		stringList{"**:0"},
		stringList{"-run", "TestAdd"},
		false,
	)
	opts.testResultPath = testPath
	opts.coverageResultPath = coveragePath

	code := runCoverage(
		ptrOptions(opts),
		[]string{dir},
		&ioStreams{stdout: io.Discard, stderr: io.Discard},
	)

	if code != successExitCode {
		t.Fatalf("exit code = %d, want 0", code)
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

func TestRunCoverageReportsTestFailuresAsLint(t *testing.T) {
	t.Parallel()

	dir := writeCLIFailingFixture(t)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	code := runCoverage(
		ptrOptions(
			optionsForTest(
				time.Minute,
				stringList{"**:0"},
				stringList{"-run", "TestFail"},
				false,
			),
		),
		[]string{dir},
		&ioStreams{stdout: &stdout, stderr: &stderr},
	)

	if code != failureExitCode {
		t.Fatalf("exit code = %d, want %d; stderr = %q", code, failureExitCode, stderr.String())
	}

	if !strings.Contains(stdout.String(), "tests failed") {
		t.Fatalf("stdout = %q, want tests failed diagnostic", stdout.String())
	}

	if !strings.Contains(stderr.String(), "coverlint: failed with") {
		t.Fatalf("stderr = %q, want failed summary", stderr.String())
	}
}

func TestReportCoveragePrintsDiagnostics(t *testing.T) {
	t.Parallel()

	runResult := coverlint.Run{
		Report: coverlint.Report{
			Results: []coverlint.Result{{
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

	code := reportCoverage(&ioStreams{stdout: &stdout, stderr: &stderr}, &runResult)

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

	failedRun := coverlint.Run{
		Report: coverlint.Report{
			Results: []coverlint.Result{{
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

	code := reportCoverage(&ioStreams{stdout: failingWriter{}, stderr: io.Discard}, &failedRun)

	if code != usageExitCode {
		t.Fatalf("diagnostic write exit code = %d, want usage exit", code)
	}

	passed := passedRunForTest()
	code = reportCoverage(&ioStreams{stdout: io.Discard, stderr: failingWriter{}}, &passed)

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

	for _, want := range []string{"Usage: coverlint", examplesHeader, "-" + ruleFlag} {
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

	disabled := optionsForTest(0, nil, nil, false)
	passed := passedRunForTest()
	code := openWebIfRequested(
		&disabled,
		&passed,
		&ioStreams{stdout: io.Discard, stderr: io.Discard},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 when web disabled", code)
	}

	var stderr bytes.Buffer

	enabled := optionsForTest(time.Minute, nil, nil, true)
	passedWeb := passedRunForTest()
	code = openWebIfRequested(
		&enabled,
		&passedWeb,
		&ioStreams{stdout: io.Discard, stderr: &stderr},
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

	values := list.Values()
	if len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("Values() = %#v", values)
	}

	values[0] = "mutated"

	if list[0] != "one" {
		t.Fatal("Values() must return a copy")
	}
}

func TestRunUsesDefaultStreams(t *testing.T) {
	code := Run([]string{"-version"})

	if code != successExitCode {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestRuleFormatErrorHelpers(t *testing.T) {
	t.Parallel()

	err := ruleFormatError{index: 1, value: "bad"}

	if got := err.Index(); got != 1 {
		t.Fatalf("Index() = %d, want 1", got)
	}

	if got := err.Value(); got != "bad" {
		t.Fatalf("Value() = %q, want bad", got)
	}

	if !errors.Is(err, errRuleFormat) {
		t.Fatal("Unwrap did not yield errRuleFormat")
	}

	empty := ruleFormatError{index: successExitCode, value: emptyString}

	if !errors.Is(empty, errRuleFormat) {
		t.Fatal("empty Unwrap did not yield errRuleFormat")
	}
}

func TestStdWriterWrite(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	writer := stdWriter(buf.Write)

	n, err := writer.Write([]byte("ok"))
	if err != nil || n != 2 {
		t.Fatalf("Write = %d, %v", n, err)
	}

	failing := stdWriter(func([]byte) (int, error) { return 0, errWriteFailed })

	_, err = failing.Write([]byte("x"))
	if err == nil || !errors.Is(err, errWriteFailed) {
		t.Fatalf("error = %v, want write failed", err)
	}
}

func TestFlagsExitCodeUsageError(t *testing.T) {
	t.Parallel()

	code := runCLI([]string{"-bad"}, &ioStreams{stdout: io.Discard, stderr: failingWriter{}})

	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}
}

func TestParseRulePartsRejectsEmptyPattern(t *testing.T) {
	t.Parallel()

	_, err := parseRuleParts(&ruleParts{
		index:       1,
		value:       ":0.5",
		pattern:     "  ",
		minimumText: "0.5",
	})
	if err == nil {
		t.Fatal("expected empty pattern error")
	}

	if !errors.Is(err, errRuleFormat) {
		t.Fatalf("error = %v, want rule format", err)
	}
}

func TestPrintUsageErrorWriteFailure(t *testing.T) {
	t.Parallel()

	code := printUsageError(failingWriter{}, errBoom)

	if code != usageExitCode {
		t.Fatalf("exit code = %d, want usage exit", code)
	}
}

func TestRunCLIRunsCoverage(t *testing.T) {
	t.Parallel()

	dir := writeCLIFixture(t)

	var stderr bytes.Buffer

	code := runCLI(
		[]string{"-rule=**:0.90", "-test-arg=-run", "-test-arg=TestAdd", dir},
		&ioStreams{stdout: io.Discard, stderr: &stderr},
	)

	if code != successExitCode {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
}

func TestFinishCoverageRunStopsOnFailure(t *testing.T) {
	t.Parallel()

	failed := coverlint.Run{
		Report: coverlint.Report{
			Results: []coverlint.Result{{
				ImportPath: "example.com/pkg",
				File:       "",
				Rule:       nil,
				Coverage:   0,
				Statements: 1,
				Covered:    0,
				Skipped:    false,
				Violation:  true,
				Message:    "below minimum",
			}},
			Checked: 1,
			Failed:  1,
			Skipped: 0,
		},
	}

	code := finishCoverageRun(
		ptrOptions(optionsForTest(time.Minute, nil, nil, true)),
		&failed,
		&ioStreams{stdout: io.Discard, stderr: io.Discard},
	)

	if code != failureExitCode {
		t.Fatalf("exit code = %d, want failure", code)
	}
}

func TestWriteFailedSummaryWriteError(t *testing.T) {
	t.Parallel()

	err := writeFailedSummary(failingWriter{}, &coverlint.Report{
		Results: nil,
		Checked: 1,
		Failed:  1,
		Skipped: 0,
	})
	if err == nil {
		t.Fatal("expected write error")
	}
}

func TestOpenWebIfRequestedSucceeds(t *testing.T) {
	t.Setenv("BROWSER", "true")

	dir := writeCLIFixture(t)

	runResult, err := coverlint.Check(
		t.Context(),
		&coverlint.Config{
			Rules:    []coverlint.Rule{{Pattern: "**", Min: 0.90}},
			Packages: nil,
			Timeout:  time.Minute.String(),
			TestArgs: []string{"-run", "TestAdd"},
		},
		dir,
	)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	code := openWebIfRequested(
		ptrOptions(optionsForTest(time.Minute, nil, nil, true)),
		&runResult,
		&ioStreams{stdout: io.Discard, stderr: io.Discard},
	)

	if code != successExitCode {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestFinishCoverageRunOpensWeb(t *testing.T) {
	t.Setenv("BROWSER", "true")

	dir := writeCLIFixture(t)

	runResult, err := coverlint.Check(
		t.Context(),
		&coverlint.Config{
			Rules:    []coverlint.Rule{{Pattern: "**", Min: 0.90}},
			Packages: nil,
			Timeout:  time.Minute.String(),
			TestArgs: []string{"-run", "TestAdd"},
		},
		dir,
	)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	code := finishCoverageRun(
		ptrOptions(optionsForTest(time.Minute, nil, nil, true)),
		&runResult,
		&ioStreams{stdout: io.Discard, stderr: io.Discard},
	)

	if code != successExitCode {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestParseRulesUsesGlobPattern(t *testing.T) {
	t.Parallel()

	rules, err := parseRules([]string{"**/internal/**:0.85", "**/critical/**:0.95"})
	if err != nil {
		t.Fatalf("parseRules: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}

	assertRule(t, rules[0], "**/internal/**", 0.85)
	assertRule(t, rules[1], "**/critical/**", 0.95)
}

func assertRule(
	t *testing.T,
	rule coverlint.Rule,
	pattern string,
	minimum float64,
) {
	t.Helper()

	if rule.Pattern != pattern {
		t.Fatalf("rule pattern = %q, want %q", rule.Pattern, pattern)
	}

	if rule.Min != minimum {
		t.Fatalf("rule minimum = %v, want %v", rule.Min, minimum)
	}
}

func TestParseRulesRejectsBadFormat(t *testing.T) {
	t.Parallel()

	rules, err := parseRules([]string{"invalid"})
	if err == nil {
		t.Fatal("parseRules succeeded, want error")
	}

	want := `parseRules: rule 1 "invalid" must have the form pattern:min`

	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}

	if rules != nil {
		t.Fatalf("rules = %v, want nil", rules)
	}

	if !errors.Is(err, errRuleFormat) {
		t.Fatalf("errors.Is(%v, errRuleFormat) = false", err)
	}
}

func TestParseRulesRejectsInvalidMinimum(t *testing.T) {
	t.Parallel()

	rules, err := parseRules([]string{"**:nope"})

	if err == nil || !strings.Contains(err.Error(), "invalid minimum") {
		t.Fatalf("error = %v, want invalid minimum", err)
	}

	if rules != nil {
		t.Fatalf("rules = %v, want nil", rules)
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
	timeout time.Duration,
	rules stringList,
	testArgs stringList,
	web bool,
) options {
	return options{
		rules:       rules,
		timeout:     timeout,
		testArgs:    testArgs,
		web:         web,
		showVersion: false,
	}
}

func passedRunForTest() coverlint.Run {
	return coverlint.Run{
		Report: coverlint.Report{
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

func writeCLIFailingFixture(t *testing.T) string {
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

func TestFail(t *testing.T) {
	t.Fatal("intentional failure")
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

func ptrOptions(opts options) *options {
	return &opts
}
