// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain_test

import (
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
