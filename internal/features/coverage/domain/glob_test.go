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
				{Pattern: test.pattern, Min: 80},
			}, nil)
			if err != nil {
				t.Fatalf("NewPolicy(%q): %v", test.pattern, err)
			}

			report := policy.Evaluate(
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
				{Pattern: pattern, Min: 80},
			}, nil)
			if err == nil {
				t.Fatalf("NewPolicy(%q) succeeded, want error", pattern)
			}
		})
	}
}

func TestPolicyUsesGlobPatterns(t *testing.T) {
	t.Parallel()

	policy, err := domain.NewPolicy([]domain.Rule{
		{Pattern: criticalPattern, Min: 95},
		{Pattern: "**/internal/**", Min: 85},
		{Pattern: "**", Min: 75},
	}, []string{"**/generated/**"})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	excludedReport := policy.Evaluate(
		[]domain.Package{testPackage("github.com/acme/project/generated/client")},
		testBlocks("github.com/acme/project/generated/client"),
	)
	if !excludedReport.Results[0].Skipped {
		t.Fatal("generated package was not excluded")
	}

	generatorReport := policy.Evaluate(
		[]domain.Package{testPackage("github.com/acme/project/generator")},
		testBlocks("github.com/acme/project/generator"),
	)
	if generatorReport.Results[0].Skipped {
		t.Fatal("non-generated package was excluded")
	}

	criticalReport := policy.Evaluate(
		[]domain.Package{testPackage("github.com/acme/project/internal/critical/http")},
		testBlocks("github.com/acme/project/internal/critical/http"),
	)
	if criticalReport.Results[0].Rule == nil || criticalReport.Results[0].Rule.Min != 95 {
		t.Fatalf("critical result = %#v, want 95%% rule", criticalReport.Results[0])
	}

	internalReport := policy.Evaluate(
		[]domain.Package{testPackage("github.com/acme/project/internal/orders")},
		testBlocks("github.com/acme/project/internal/orders"),
	)
	if internalReport.Results[0].Rule == nil || internalReport.Results[0].Rule.Min != 85 {
		t.Fatalf("internal result = %#v, want 85%% rule", internalReport.Results[0])
	}

	defaultReport := policy.Evaluate(
		[]domain.Package{testPackage("github.com/acme/project/api")},
		testBlocks("github.com/acme/project/api"),
	)
	if defaultReport.Results[0].Rule == nil || defaultReport.Results[0].Rule.Min != 75 {
		t.Fatalf("default result = %#v, want 75%% rule", defaultReport.Results[0])
	}
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
