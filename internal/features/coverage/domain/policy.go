package domain

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	percentageMultiplier      = 100
	coverageComparisonEpsilon = 1e-9
)

var (
	errMissingCoverageRule = errors.New("at least one coverage rule is required")
	errMissingRulePattern  = errors.New("pattern is required")
	errInvalidMinimum      = errors.New("min must be finite, greater than 0, and at most 100")
)

type compiledRule struct {
	rule Rule
	glob globPattern
}

// Policy evaluates coverage blocks against ordered rules and excludes.
type Policy struct {
	rules    []compiledRule
	excludes []globPattern
}

// NewPolicy compiles ordered coverage rules and exclude patterns.
func NewPolicy(rules []Rule, excludes []string) (Policy, error) {
	if len(rules) == 0 {
		return Policy{}, errMissingCoverageRule
	}

	compiled := make([]compiledRule, 0, len(rules))
	for index, rule := range rules {
		if rule.Pattern == "" {
			return Policy{}, fmt.Errorf("override %d: %w", index+1, errMissingRulePattern)
		}

		err := validateMinimum(rule.Min)
		if err != nil {
			return Policy{}, fmt.Errorf("override %d: %w", index+1, err)
		}

		glob, err := compileGlob(rule.Pattern)
		if err != nil {
			return Policy{}, fmt.Errorf(
				"override %d: invalid glob %q: %w",
				index+1,
				rule.Pattern,
				err,
			)
		}

		compiled = append(compiled, compiledRule{rule: rule, glob: glob})
	}

	compiledExcludes := make([]globPattern, 0, len(excludes))
	for i, pattern := range excludes {
		glob, err := compileGlob(pattern)
		if err != nil {
			return Policy{}, fmt.Errorf("exclude %d: invalid glob %q: %w", i+1, pattern, err)
		}

		compiledExcludes = append(compiledExcludes, glob)
	}

	return Policy{
		rules:    compiled,
		excludes: compiledExcludes,
	}, nil
}

// Evaluate applies the policy to packages and coverage blocks.
func (p Policy) Evaluate(packages []Package, blocks []Block) Report {
	stats := aggregate(packages, blocks)
	report := Report{
		Results: make([]Result, 0, len(packages)),
		Checked: 0,
		Failed:  0,
		Skipped: 0,
	}

	for _, pkg := range packages {
		result := p.evaluatePackage(pkg, stats[pkg.ImportPath])
		report.add(result)
		report.Results = append(report.Results, result)
	}

	return report
}

func (p Policy) evaluatePackage(pkg Package, item packageStats) Result {
	result := newResult(pkg)
	if p.excluded(pkg.ImportPath) {
		return skippedResult(result, fmt.Sprintf("package %q is excluded", pkg.ImportPath))
	}

	rule := p.match(pkg.ImportPath)
	if rule == nil {
		return skippedResult(
			result,
			fmt.Sprintf("package %q has no coverage policy", pkg.ImportPath),
		)
	}

	ruleCopy := rule.rule
	result.Rule = &ruleCopy
	result.Covered = item.covered
	result.Statements = item.statements

	switch {
	case len(pkg.Files) == 0:
		return skippedResult(
			result,
			fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath),
		)
	case item.blocks == 0:
		return skippedResult(
			result,
			fmt.Sprintf("package %q has no coverage profile blocks", pkg.ImportPath),
		)
	case item.statements == 0:
		return skippedResult(
			result,
			fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath),
		)
	}

	result.Coverage = float64(item.covered) * percentageMultiplier / float64(item.statements)
	result.Violation = result.Coverage+coverageComparisonEpsilon < rule.rule.Min
	result.Message = coverageMessage(result, rule.rule.Min)

	return result
}

func newResult(pkg Package) Result {
	return Result{
		ImportPath: pkg.ImportPath,
		File:       pkg.FirstFile,
		Rule:       nil,
		Coverage:   0,
		Statements: 0,
		Covered:    0,
		Skipped:    false,
		Violation:  false,
		Message:    "",
	}
}

func skippedResult(result Result, message string) Result {
	result.Skipped = true
	result.Message = message

	return result
}

func coverageMessage(result Result, minimum float64) string {
	format := "coverage %.2f%% meets %.2f%% for package %q (%d/%d statements)"
	if result.Violation {
		format = "coverage %.2f%% is below %.2f%% for package %q (%d/%d statements)"
	}

	return fmt.Sprintf(
		format,
		result.Coverage,
		minimum,
		result.ImportPath,
		result.Covered,
		result.Statements,
	)
}

func (r *Report) add(result Result) {
	switch {
	case result.Skipped:
		r.Skipped++
	case result.Violation:
		r.Checked++
		r.Failed++
	default:
		r.Checked++
	}
}

type packageStats struct {
	covered    int64
	statements int64
	blocks     int
}

type packageIndex struct {
	files map[string]fileMatch
}

type fileMatch struct {
	importPath string
	file       string
}

type blockKey struct {
	importPath string
	file       string
	position   string
}

type mergedBlock struct {
	statements int64
	covered    bool
}

func aggregate(packages []Package, blocks []Block) map[string]packageStats {
	index := newPackageIndex(packages)
	stats := make(map[string]packageStats, len(packages))
	merged := make(map[blockKey]mergedBlock, len(blocks))

	for _, block := range blocks {
		match := index.lookup(block.File)
		if match.importPath == "" {
			continue
		}

		key := blockKey{
			importPath: match.importPath,
			file:       match.file,
			position:   block.Position,
		}
		item := merged[key]
		item.statements = block.Statements
		item.covered = item.covered || block.Covered
		merged[key] = item
	}

	for key, block := range merged {
		if block.statements == 0 {
			continue
		}

		item := stats[key.importPath]
		item.blocks++

		item.statements += block.statements
		if block.covered {
			item.covered += block.statements
		}

		stats[key.importPath] = item
	}

	return stats
}

func newPackageIndex(packages []Package) packageIndex {
	cwd, _ := os.Getwd()
	index := packageIndex{
		files: make(map[string]fileMatch),
	}

	for _, pkg := range packages {
		for _, filename := range pkg.Files {
			absolute := filename
			if !filepath.IsAbs(absolute) && pkg.Dir != "" {
				absolute = filepath.Join(pkg.Dir, absolute)
			}

			absolute = normalizePath(absolute)
			match := fileMatch{
				importPath: pkg.ImportPath,
				file:       absolute,
			}
			index.files[absolute] = match

			relative, err := filepath.Rel(
				cwd,
				filepath.FromSlash(absolute),
			)
			if err == nil && isLocalRelative(relative) {
				index.files[normalizePath(relative)] = match
			}

			if pkg.Dir != "" {
				relative, err = filepath.Rel(
					pkg.Dir,
					filepath.FromSlash(absolute),
				)
				if err == nil && isLocalRelative(relative) {
					index.files[normalizePath(path.Join(pkg.ImportPath, filepath.ToSlash(relative)))] = match
				}
			}
		}
	}

	return index
}

func (i packageIndex) lookup(filename string) fileMatch {
	filename = normalizePath(filename)

	return i.files[filename]
}

// ValidateMinimum checks whether a coverage minimum is allowed.
func ValidateMinimum(value float64) error {
	return validateMinimum(value)
}

func validateMinimum(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > percentageMultiplier {
		return fmt.Errorf("%w, got %.2f", errInvalidMinimum, value)
	}

	return nil
}

func (p Policy) excluded(importPath string) bool {
	for _, glob := range p.excludes {
		if glob.Match(importPath) {
			return true
		}
	}

	return false
}

func (p Policy) match(importPath string) *compiledRule {
	for i := range p.rules {
		if p.rules[i].glob.Match(importPath) {
			return &p.rules[i]
		}
	}

	return nil
}

func isLocalRelative(value string) bool {
	return value != "." && value != "" && value != ".." &&
		!strings.HasPrefix(value, ".."+string(filepath.Separator))
}

func normalizePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))

	return strings.TrimPrefix(value, "./")
}
