// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

var errBoom = errors.New("boom")

func TestCompileGlobEmptyPattern(t *testing.T) {
	t.Parallel()

	_, err := compileGlob("")
	if !errors.Is(err, errEmptyPattern) {
		t.Fatalf("error = %v, want empty pattern", err)
	}
}

func TestGlobMatchUsesMemoizedVisitedState(t *testing.T) {
	t.Parallel()

	matcher := newGlobMatcher(globPattern{segments: []string{"**", "**"}}, "a/b/c")
	if !globMatch(matcher, zero, zero) {
		t.Fatal("expected match")
	}

	if !globMatch(matcher, zero, zero) {
		t.Fatal("expected memoized match")
	}
}

func TestGlobMatchSegmentRejectsInvalidPattern(t *testing.T) {
	t.Parallel()

	matcher := &globMatcher{
		segments:     []string{"["},
		pathSegments: []string{"a"},
		memo:         make(map[[indexPairSize]int]bool),
		visited:      make(map[[indexPairSize]int]bool),
	}

	if globMatchSegment(matcher, zero, zero) {
		t.Fatal("expected invalid segment match to fail")
	}
}

func TestNewPolicyRejectsEmptyRulesAndBadExcludes(t *testing.T) {
	t.Parallel()

	_, err := NewPolicy(nil, nil)
	if !errors.Is(err, errMissingCoverageRule) {
		t.Fatalf("error = %v, want missing coverage rule", err)
	}

	_, err = NewPolicy([]Rule{{Pattern: "**", Min: 0.80}}, []string{"["})
	if err == nil || !strings.Contains(err.Error(), "compile coverage excludes:") {
		t.Fatalf("error = %v, want exclude compile error", err)
	}
}

func TestRuleMatchesAndString(t *testing.T) {
	t.Parallel()

	rule := Rule{Pattern: "github.com/acme/**", Min: 0.85}

	if !rule.Matches("github.com/acme/orders") {
		t.Fatal("expected match")
	}

	if (Rule{Pattern: "[", Min: 0.80}).Matches("anything") {
		t.Fatal("invalid glob should not match")
	}

	if got := rule.String(); got != "github.com/acme/**:0.85" {
		t.Fatalf("String = %q", got)
	}
}

func TestAddMergedBlockSkipsZeroStatements(t *testing.T) {
	t.Parallel()

	stats := map[string]packageStats{}
	key := blockKey{"pkg", "file\x00pos"}

	addMergedBlock(stats, &key, &mergedBlock{statements: zero, covered: true})

	if len(stats) != 0 {
		t.Fatalf("stats = %#v, want empty", stats)
	}
}

func TestAddRelativeIndexFileStoresLocalRelative(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "file.go")
	index := packageIndex{files: make(map[string]fileMatch)}
	match := fileMatch{importPath: "example.com/pkg", file: normalizePath(file)}

	addRelativeIndexFile(index, &match, &relativePaths{cwd: dir, absolute: normalizePath(file)})

	relative := normalizePath("file.go")
	if _, ok := index.files[relative]; !ok {
		t.Fatalf("files = %#v, want relative entry %q", index.files, relative)
	}
}

func TestAddReportCountsViolation(t *testing.T) {
	t.Parallel()

	report := Report{}
	addReport(&report, &Result{Violation: true})

	if report.Checked != 1 || report.Failed != 1 || report.Skipped != 0 {
		t.Fatalf("report = %#v, want checked+failed", report)
	}
}

func TestCompileExcludesRejectsInvalidGlob(t *testing.T) {
	t.Parallel()

	_, err := compileExcludes([]string{"**", "["})
	if err == nil || !strings.Contains(err.Error(), "exclude 2:") {
		t.Fatalf("error = %v, want exclude index error", err)
	}
}

func TestCoverageMessageAndSkipReason(t *testing.T) {
	t.Parallel()

	pass := &Result{
		ImportPath: "example.com/pkg",
		Coverage:   90,
		Covered:    9,
		Statements: 10,
		Violation:  false,
	}
	if !strings.Contains(coverageMessage(pass, 80), "meets") {
		t.Fatalf("message = %q, want meets", coverageMessage(pass, 80))
	}

	fail := &Result{
		ImportPath: "example.com/pkg",
		Coverage:   50,
		Covered:    5,
		Statements: 10,
		Violation:  true,
	}
	if !strings.Contains(coverageMessage(fail, 80), "is below") {
		t.Fatalf("message = %q, want below", coverageMessage(fail, 80))
	}

	pkg := &Package{ImportPath: "example.com/pkg", Files: []string{"file.go"}}
	reason := coverageSkipReason(pkg, packageStats{statements: 1, blocks: zero})
	if !strings.Contains(reason, "has no coverage profile blocks") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestNewPackageIndexGetwdFailure(t *testing.T) {
	previous := getwd

	t.Cleanup(func() { getwd = previous })

	getwd = func() (string, error) { return "", errBoom }

	index := newPackageIndex([]Package{{
		ImportPath: "example.com/pkg",
		Dir:        "",
		Files:      []string{"file.go"},
		FirstFile:  "file.go",
	}})

	if _, ok := index.files[normalizePath("file.go")]; !ok {
		t.Fatalf("files = %#v, want normalized relative file", index.files)
	}
}

func TestPackageFilePathJoinsRelativeFilename(t *testing.T) {
	t.Parallel()

	got := packageFilePath(&Package{Dir: "/repo/pkg"}, "file.go")
	want := normalizePath(filepath.Join("/repo/pkg", "file.go"))

	if got != want {
		t.Fatalf("packageFilePath = %q, want %q", got, want)
	}
}

func TestEvaluateReportsViolation(t *testing.T) {
	t.Parallel()

	policy, err := NewPolicy([]Rule{{Pattern: "**", Min: 1.0}}, nil)
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "file.go")
	pkg := Package{
		ImportPath: "example.com/pkg",
		Dir:        dir,
		Files:      []string{file},
		FirstFile:  file,
	}

	report := Evaluate(&policy, []Package{pkg}, []Block{{
		File:       file,
		Position:   "1.1,2.2",
		Statements: 2,
		Covered:    false,
	}})

	if report.Failed != 1 || report.Checked != 1 {
		t.Fatalf("report = %#v, want violation", report)
	}

	if !strings.Contains(report.Results[0].Message, "is below") {
		t.Fatalf("message = %q", report.Results[0].Message)
	}
}
