package domain

import "testing"

func TestGlobPatternMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pattern    string
		importPath string
		want       bool
	}{
		{name: "all packages", pattern: "**", importPath: "github.com/acme/project", want: true},
		{name: "recursive exact package", pattern: "**/internal/critical/**", importPath: "github.com/acme/project/internal/critical", want: true},
		{name: "recursive child package", pattern: "**/internal/critical/**", importPath: "github.com/acme/project/internal/critical/http", want: true},
		{name: "recursive does not match prefix", pattern: "**/internal/critical/**", importPath: "github.com/acme/project/internal/criticality", want: false},
		{name: "single segment wildcard", pattern: "github.com/acme/*/orders", importPath: "github.com/acme/project/orders", want: true},
		{name: "single segment wildcard does not cross slash", pattern: "github.com/acme/*/orders", importPath: "github.com/acme/team/project/orders", want: false},
		{name: "question mark", pattern: "**/mock?", importPath: "github.com/acme/project/mocks", want: true},
		{name: "character class", pattern: "**/[io]rders", importPath: "github.com/acme/project/orders", want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			glob, err := compileGlob(test.pattern)
			if err != nil {
				t.Fatalf("compileGlob(%q): %v", test.pattern, err)
			}
			if got := glob.Match(test.importPath); got != test.want {
				t.Fatalf("glob.Match(%q) = %v, want %v", test.importPath, got, test.want)
			}
		})
	}
}

func TestCompileGlobRejectsInvalidPatterns(t *testing.T) {
	t.Parallel()

	for _, pattern := range []string{"", "/internal/**", "internal//**", "internal/**suffix", "internal/["} {
		pattern := pattern
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()
			if _, err := compileGlob(pattern); err == nil {
				t.Fatalf("compileGlob(%q) succeeded, want error", pattern)
			}
		})
	}
}

func TestPolicyUsesGlobPatterns(t *testing.T) {
	t.Parallel()

	policy, err := NewPolicy([]Rule{
		{Pattern: "**/internal/critical/**", Min: 95},
		{Pattern: "**/internal/**", Min: 85},
		{Pattern: "**", Min: 75},
	}, []string{"**/generated/**"})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	if !policy.excluded("github.com/acme/project/generated/client") {
		t.Fatal("generated package was not excluded")
	}
	if policy.excluded("github.com/acme/project/generator") {
		t.Fatal("non-generated package was excluded")
	}

	rule := policy.match("github.com/acme/project/internal/critical/http")
	if rule == nil || rule.rule.Min != 95 {
		t.Fatalf("critical package matched %#v, want 95%% rule", rule)
	}

	rule = policy.match("github.com/acme/project/internal/orders")
	if rule == nil || rule.rule.Min != 85 {
		t.Fatalf("internal package matched %#v, want 85%% rule", rule)
	}

	rule = policy.match("github.com/acme/project/api")
	if rule == nil || rule.rule.Min != 75 {
		t.Fatalf("default package matched %#v, want 75%% rule", rule)
	}
}
