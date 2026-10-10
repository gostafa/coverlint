// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTestReport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, input, pkg, test, detail string
		failed                         bool
	}{
		{name: "passing", input: "ok\texample.com/pkg\t0.1s\n"},
		{
			name:  "ordinary log",
			input: "failure expected\n    FAIL is only a word\nok\texample.com/pkg\t0.1s\n",
		},
		{name: "empty"},
		{name: "blank", input: "\n\n"},
		{
			name:   "package",
			input:  "panic: broken\nFAIL\texample.com/pkg\t0.1s\n",
			pkg:    "example.com/pkg",
			detail: "panic: broken",
			failed: true,
		},
		{name: "standalone", input: "FAIL\n", detail: "FAIL", failed: true},
		{
			name:   "unattributed",
			input:  "--- FAIL: TestCalc (0.00s)\n    calc_test.go:9: got=1 want=2\n",
			test:   "TestCalc",
			detail: "got=1 want=2",
			failed: true,
		},
		{
			name:   "subtest",
			input:  "--- FAIL: TestCalc/case (0.00s)\n    calc_test.go:9: got=1 want=2\nFAIL\nFAIL\texample.com/pkg\t0.1s\n",
			pkg:    "example.com/pkg",
			test:   "TestCalc/case",
			detail: "got=1 want=2",
			failed: true,
		},
		{
			name:   "JSON",
			input:  "{\"Action\":\"output\",\"Package\":\"example.com/pkg\",\"Test\":\"TestCalc\",\"Output\":\"got=1 want=2\\n\"}\n{\"Action\":\"fail\",\"Package\":\"example.com/pkg\",\"Test\":\"TestCalc\"}",
			pkg:    "example.com/pkg",
			test:   "TestCalc",
			detail: "got=1 want=2",
			failed: true,
		},
		{
			name:  "passing JSON",
			input: "{\"Action\":\"output\",\"Output\":\"FAIL is expected\\n\"}\n{\"Action\":\"pass\",\"Package\":\"example.com/pkg\"}\n",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			report, err := parseTestReport(
				bufio.NewScanner(strings.NewReader(testCase.input)),
				"report.txt",
			)
			if err != nil || report.TestsFailed != testCase.failed {
				t.Fatalf("report=%#v err=%v", report, err)
			}
			if !testCase.failed {
				return
			}
			if len(report.Failures) != 1 {
				t.Fatalf("failures=%#v", report.Failures)
			}
			failure := report.Failures[0]
			if failure.ImportPath != testCase.pkg || failure.Test != testCase.test ||
				!strings.Contains(failure.Output, testCase.detail) ||
				!strings.Contains(failure.Output, "report.txt") {
				t.Fatalf("failure=%#v", failure)
			}
		})
	}
}

func TestReadTestReportOptionalAndErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, path := range []string{"", filepath.Join(dir, "missing.txt")} {
		report, err := ReadTestReport(path)
		if err != nil || report.TestsFailed {
			t.Fatalf("path=%q report=%#v err=%v", path, report, err)
		}
	}
	if _, err := ReadTestReport(dir); err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("directory error=%v", err)
	}
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte("FAIL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ReadTestReport(path)
	if err != nil || !report.TestsFailed {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if _, err := ReadTestReport(
		filepath.Join(path, "child"),
	); err == nil ||
		!strings.Contains(err.Error(), "open test report") {
		t.Fatalf("invalid parent error=%v", err)
	}
}

func TestTextReportPackageGrouping(t *testing.T) {
	t.Parallel()
	input := "--- FAIL: TestFirst (0s)\n    first assertion\nFAIL\nFAIL\ta/pkg\t0.1s\n--- PASS: TestGood (0s)\nok\tb/pkg\t0.1s\n--- FAIL: TestLast (0s)\n    last assertion\nFAIL\tc/pkg\t0.1s\nFAIL\n"
	report, err := parseTestReport(bufio.NewScanner(strings.NewReader(input)), "grouped.txt")
	if err != nil || len(report.Failures) != 2 || len(report.FailedPackages) != 2 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if report.Failures[0].ImportPath != "a/pkg" || report.Failures[1].ImportPath != "c/pkg" {
		t.Fatalf("failures=%#v", report.Failures)
	}
}

func TestTextReportTruncationKeepsNames(t *testing.T) {
	t.Parallel()
	input := strings.Repeat(
		"ordinary output\n",
		commandOutputLimit/10,
	) + "--- FAIL: TestLate (0s)\n    assertion\nFAIL\tpkg\t0.1s\n"
	report, err := parseTestReport(bufio.NewScanner(strings.NewReader(input)), "large.txt")
	if err != nil || len(report.Failures) != 1 || report.Failures[0].Test != "TestLate" ||
		!strings.Contains(report.Failures[0].Output, "truncated") {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if name := textFailureName(textTestFailPrefix); name != "" {
		t.Fatalf("empty name=%q", name)
	}
}
