// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain_test

import (
	"math"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const criticalPattern = "**/internal/critical/**"

func TestGlobPatternMatch(t *testing.T) {
	t.Parallel()

	tests := globMatchTests()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			policy, err := domain.NewPolicy([]domain.Rule{
				{Pattern: test.pattern, Min: 0.80},
			})
			if err != nil {
				t.Fatalf("NewPolicy(%q): %v", test.pattern, err)
			}

			report := domain.Evaluate(
				&policy,
				[]domain.Package{testPackage(test.importPath)},
				testBlocks(test.importPath),
			)

			got := report.Checked == 1

			if got != test.want {
				t.Fatalf("policy matched %q = %v, want %v", test.importPath, got, test.want)
			}
		})
	}
}

func globMatchTests() []struct {
	name       string
	pattern    string
	importPath string
	want       bool
} {
	tests := []struct {
		name       string
		pattern    string
		importPath string
		want       bool
	}{
		{name: "all packages", pattern: "**", importPath: "github.com/acme/project", want: true},
		{
			name:       "recursive exact package",
			pattern:    criticalPattern,
			importPath: "github.com/acme/project/internal/critical",
			want:       true,
		},
		{
			name:       "recursive child package",
			pattern:    criticalPattern,
			importPath: "github.com/acme/project/internal/critical/http",
			want:       true,
		},
		{
			name:       "recursive does not match prefix",
			pattern:    criticalPattern,
			importPath: "github.com/acme/project/internal/criticality",
			want:       false,
		},
		{
			name:       "single segment wildcard",
			pattern:    "github.com/acme/*/orders",
			importPath: "github.com/acme/project/orders",
			want:       true,
		},
		{
			name:       "single segment wildcard does not cross slash",
			pattern:    "github.com/acme/*/orders",
			importPath: "github.com/acme/team/project/orders",
			want:       false,
		},
		{
			name:       "question mark",
			pattern:    "**/mock?",
			importPath: "github.com/acme/project/mocks",
			want:       true,
		},
		{
			name:       "character class",
			pattern:    "**/[io]rders",
			importPath: "github.com/acme/project/orders",
			want:       true,
		},
	}

	return tests
}

func TestCompileGlobRejectsInvalidPatterns(t *testing.T) {
	t.Parallel()

	for _, pattern := range []string{"", "/internal/**", "internal//**", "internal/**suffix", "internal/["} {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewPolicy([]domain.Rule{
				{Pattern: pattern, Min: 0.80},
			})
			if err == nil {
				t.Fatalf("NewPolicy(%q) succeeded, want error", pattern)
			}
		})
	}
}

func TestPolicyUsesGlobPatterns(t *testing.T) {
	t.Parallel()

	policy, err := domain.NewPolicy([]domain.Rule{
		{Pattern: criticalPattern, Min: 0.95},
		{Pattern: "**/internal/**", Min: 0.85},
		{Pattern: "**", Min: 0.75},
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	for _, test := range policyGlobTests() {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertPolicyGlobResult(t, policy, test.importPath, test.minimum)
		})
	}
}

func policyGlobTests() []struct {
	name       string
	importPath string
	minimum    float64
} {
	return []struct {
		name       string
		importPath string
		minimum    float64
	}{
		{
			name:       "generator",
			importPath: "github.com/acme/project/generator",
			minimum:    0.75,
		},
		{
			name:       "critical",
			importPath: "github.com/acme/project/internal/critical/http",
			minimum:    0.95,
		},
		{
			name:       "internal",
			importPath: "github.com/acme/project/internal/orders",
			minimum:    0.85,
		},
		{
			name:       "default",
			importPath: "github.com/acme/project/api",
			minimum:    0.75,
		},
	}
}

func assertPolicyGlobResult(
	t *testing.T,
	policy domain.Policy,
	importPath string,
	minimum float64,
) {
	t.Helper()

	report := domain.Evaluate(
		&policy,
		[]domain.Package{testPackage(importPath)},
		testBlocks(importPath),
	)

	result := report.Results[0]

	if result.Skipped {
		t.Fatalf("result = %#v, want not skipped", result)
	}

	if result.Rule == nil {
		t.Fatalf("result = %#v, want rule", result)
	}

	if result.Rule.Min != minimum {
		t.Fatalf("rule minimum = %v, want %v", result.Rule.Min, minimum)
	}
}

func TestPolicySelectsMostSpecificRule(t *testing.T) {
	t.Parallel()

	policy, err := domain.NewPolicy([]domain.Rule{
		{Pattern: "**", Min: 0.80},
		{Pattern: "**/*_test", Min: 0},
		{Pattern: "**/internal/**", Min: 0.20},
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	for _, test := range []struct {
		name       string
		importPath string
		minimum    float64
	}{
		{name: "catch-all", importPath: "github.com/acme/project/api", minimum: 0.80},
		{name: "internal", importPath: "github.com/acme/project/internal/orders", minimum: 0.20},
		{name: "test package", importPath: "github.com/acme/project/pkg_test", minimum: 0},
		{
			name:       "internal test package",
			importPath: "github.com/acme/project/internal/foo_test",
			minimum:    0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertPolicyGlobResult(t, policy, test.importPath, test.minimum)
		})
	}
}

func TestPolicyLaterRuleWinsExactSpecificityTie(t *testing.T) {
	t.Parallel()

	policy, err := domain.NewPolicy([]domain.Rule{
		{Pattern: "**/orders", Min: 0.80},
		{Pattern: "**/orders", Min: 0.50},
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	assertPolicyGlobResult(t, policy, "github.com/acme/project/orders", 0.50)
}

func testPackage(importPath string) domain.Package {
	file := importPath + "/file.go"

	return domain.Package{
		ImportPath: importPath,
		Dir:        "",
		Files:      []string{file},
		FirstFile:  file,
	}
}

func testBlocks(importPath string) []domain.Block {
	return []domain.Block{
		{
			File:       importPath + "/file.go",
			Position:   "1.1,2.2",
			Statements: 1,
			Covered:    true,
		},
	}
}

const (
	ordersFile    = "/repo/internal/orders/orders.go"
	ordersPackage = "github.com/acme/project/internal/orders"
	blockPosition = "10.1,12.2"
)

func TestNewPolicyRejectsNonFiniteMinimums(t *testing.T) {
	t.Parallel()

	for _, minimum := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1, 1.1} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewPolicy([]domain.Rule{{Pattern: "**", Min: minimum}})
			if err == nil {
				t.Fatalf("NewPolicy minimum %v succeeded, want error", minimum)
			}
		})
	}
}

func TestNewPolicyAcceptsZeroMinimum(t *testing.T) {
	t.Parallel()

	_, err := domain.NewPolicy([]domain.Rule{{Pattern: "**", Min: 0}})
	if err != nil {
		t.Fatalf("NewPolicy zero minimum: %v", err)
	}
}

func TestEvaluateDeduplicatesProfileBlocksByPosition(t *testing.T) {
	t.Parallel()

	policy := mustPolicy(t)
	report := domain.Evaluate(&policy, ordersPackagesForTest(), duplicateBlocksForTest())

	assertDeduplicatedReport(t, report)
}

func assertDeduplicatedReport(t *testing.T, report domain.Report) {
	t.Helper()

	assertReportChecked(t, report)
	assertDeduplicatedResult(t, report.Results[0])
}

func assertReportChecked(t *testing.T, report domain.Report) {
	t.Helper()

	if report.Failed != 0 {
		t.Fatalf("report = %#v, want no failures", report)
	}

	if report.Checked != 1 {
		t.Fatalf("report = %#v, want one checked package", report)
	}
}

func assertDeduplicatedResult(t *testing.T, result domain.Result) {
	t.Helper()

	if result.Statements != 2 {
		t.Fatalf("result = %#v, want two statements", result)
	}

	if result.Covered != 2 {
		t.Fatalf("result = %#v, want duplicate block covered", result)
	}

	if result.Coverage != 100 {
		t.Fatalf("result = %#v, want 100%% coverage", result)
	}
}

func ordersPackagesForTest() []domain.Package {
	return []domain.Package{{
		ImportPath: ordersPackage,
		Dir:        "/repo/internal/orders",
		Files:      []string{ordersFile},
		FirstFile:  ordersFile,
	}}
}

func duplicateBlocksForTest() []domain.Block {
	return []domain.Block{
		{
			File:       ordersFile,
			Position:   blockPosition,
			Statements: 2,
			Covered:    false,
		},
		{
			File:       ordersFile,
			Position:   blockPosition,
			Statements: 2,
			Covered:    true,
		},
		{
			File:       ordersPackage + "/orders.go",
			Position:   blockPosition,
			Statements: 2,
			Covered:    false,
		},
	}
}

func TestEvaluateDoesNotMapUnknownChildFileToAncestorPackage(t *testing.T) {
	t.Parallel()

	policy := mustPolicy(t)
	report := domain.Evaluate(&policy, []domain.Package{{
		ImportPath: ordersPackage,
		Dir:        "/repo/internal/orders",
		Files:      []string{ordersFile},
		FirstFile:  ordersFile,
	}}, []domain.Block{
		{
			File:       "/repo/internal/orders/subpkg/sub.go",
			Position:   "1.1,2.2",
			Statements: 1,
			Covered:    true,
		},
	})

	if report.Checked != 0 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("report = %#v, want unknown child file ignored and package skipped", report)
	}

	if !report.Results[0].Skipped {
		t.Fatalf("result = %#v, want skipped result", report.Results[0])
	}
}

func TestEvaluateSkipsPackageWithNoProfileBlocks(t *testing.T) {
	t.Parallel()

	policy := mustPolicy(t)
	report := domain.Evaluate(&policy, []domain.Package{{
		ImportPath: "github.com/acme/project/internal/contracts",
		Dir:        "/repo/internal/contracts",
		Files:      []string{"/repo/internal/contracts/contracts.go"},
		FirstFile:  "/repo/internal/contracts/contracts.go",
	}}, nil)

	if report.Checked != 0 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("report = %#v, want package skipped", report)
	}
}

func mustPolicy(t *testing.T) domain.Policy {
	t.Helper()

	policy, err := domain.NewPolicy([]domain.Rule{{Pattern: "**", Min: 0.80}})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	return policy
}

func TestAppendTestFailuresAddsPackageViolations(t *testing.T) {
	t.Parallel()

	report := domain.Report{
		Results: nil,
		Checked: 0,
		Failed:  0,
		Skipped: 0,
	}
	packages := []domain.Package{{
		ImportPath: "example.com/pkg",
		Dir:        "/repo/pkg",
		Files:      []string{"/repo/pkg/a.go"},
		FirstFile:  "/repo/pkg/a.go",
	}}
	coverage := domain.Coverage{
		Profile:        nil,
		Blocks:         nil,
		FailedPackages: []string{"example.com/pkg"},
		TestOutput:     "FAIL\texample.com/pkg\t0.01s\n",
		TestsFailed:    true,
	}

	domain.AppendTestFailures(&report, packages, &coverage)

	if report.Failed != 1 || report.Checked != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}

	result := report.Results[0]

	if !result.Violation || result.File != "/repo/pkg/a.go" {
		t.Fatalf("result = %#v", result)
	}

	if !strings.Contains(result.Message, `tests failed for package "example.com/pkg"`) {
		t.Fatalf("Message = %q", result.Message)
	}
}

func TestAppendTestFailuresAddsSyntheticViolation(t *testing.T) {
	t.Parallel()

	report := domain.Report{
		Results: nil,
		Checked: 0,
		Failed:  0,
		Skipped: 0,
	}
	coverage := domain.Coverage{
		Profile:        nil,
		Blocks:         nil,
		FailedPackages: nil,
		TestOutput:     "something broke",
		TestsFailed:    true,
	}

	domain.AppendTestFailures(&report, nil, &coverage)

	if report.Failed != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}

	if report.Results[0].Message != "tests failed\nsomething broke" {
		t.Fatalf("Message = %q", report.Results[0].Message)
	}
}

func TestAppendTestFailuresNoopWithoutFailures(t *testing.T) {
	t.Parallel()

	report := domain.Report{
		Results: nil,
		Checked: 1,
		Failed:  0,
		Skipped: 0,
	}

	domain.AppendTestFailures(&report, nil, &domain.Coverage{
		Profile:        nil,
		Blocks:         nil,
		FailedPackages: nil,
		TestOutput:     "ok\texample.com/pkg\t0.01s\n",
		TestsFailed:    false,
	})

	if report.Failed != 0 || len(report.Results) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestFailureDetailsOnlyIncludeFailedTestsForPackage(t *testing.T) {
	t.Parallel()

	coverage := domain.Coverage{
		TestsFailed:    true,
		FailedPackages: []string{"example.com/a", "example.com/b"},
		Failures: []domain.TestFailure{
			{
				ImportPath: "example.com/a",
				Test:       "TestCalc/sub",
				Output:     "    calc_test.go:12: input=2 got=3 want=4\n",
			},
			{ImportPath: "example.com/b", Test: "TestSilent"},
			{ImportPath: "example.com/a", Output: "passing log\nFAIL\n"},
		},
	}
	var report domain.Report
	domain.AppendTestFailures(&report, nil, &coverage)
	if report.Failed != 2 || len(report.Results) != 2 {
		t.Fatalf("report = %#v", report)
	}
	message := report.Results[0].Message
	if !strings.Contains(message, "TestCalc/sub\n    calc_test.go:12: input=2 got=3 want=4") {
		t.Fatalf("message = %q", message)
	}
	if strings.Contains(message, "passing log") || strings.Contains(message, "TestSilent") {
		t.Fatalf("unrelated output in message = %q", message)
	}
	if !strings.Contains(report.Results[1].Message, "TestSilent") {
		t.Fatalf("silent failure = %q", report.Results[1].Message)
	}
}

func TestFailureDetailsIncludePackageOutputAndTruncation(t *testing.T) {
	t.Parallel()

	coverage := domain.Coverage{
		TestsFailed:    true,
		FailedPackages: []string{"example.com/a"},
		Failures: []domain.TestFailure{
			{ImportPath: "example.com/a", Output: "panic: test setup failed\n"},
			{Output: "... output truncated by coverlint ..."},
		},
	}
	var report domain.Report
	domain.AppendTestFailures(&report, nil, &coverage)
	message := report.Results[0].Message
	if !strings.Contains(message, "panic: test setup failed") ||
		!strings.Contains(message, "output truncated") {
		t.Fatalf("message = %q", message)
	}
}
