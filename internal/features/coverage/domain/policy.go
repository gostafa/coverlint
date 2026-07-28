package domain

import (
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type compiledRule struct {
	rule Rule
	glob globPattern
}

type Policy struct {
	rules    []compiledRule
	excludes []globPattern
}

func NewPolicy(rules []Rule, excludes []string) (Policy, error) {
	if len(rules) == 0 {
		return Policy{}, fmt.Errorf("at least one coverage rule is required")
	}

	compiled := make([]compiledRule, 0, len(rules))
	for i, rule := range rules {
		if rule.Pattern == "" {
			return Policy{}, fmt.Errorf("override %d: pattern is required", i+1)
		}
		if err := validateMinimum(rule.Min); err != nil {
			return Policy{}, fmt.Errorf("override %d: %w", i+1, err)
		}
		glob, err := compileGlob(rule.Pattern)
		if err != nil {
			return Policy{}, fmt.Errorf("override %d: invalid glob %q: %w", i+1, rule.Pattern, err)
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

	return Policy{rules: compiled, excludes: compiledExcludes}, nil
}

func (p Policy) Evaluate(packages []Package, blocks []Block) Report {
	stats := aggregate(packages, blocks)
	report := Report{Results: make([]Result, 0, len(packages))}

	for _, pkg := range packages {
		result := Result{ImportPath: pkg.ImportPath, File: pkg.FirstFile}

		if p.excluded(pkg.ImportPath) {
			result.Skipped = true
			result.Message = fmt.Sprintf("package %q is excluded", pkg.ImportPath)
			report.Skipped++
			report.Results = append(report.Results, result)
			continue
		}

		rule := p.match(pkg.ImportPath)
		if rule == nil {
			result.Skipped = true
			result.Message = fmt.Sprintf("package %q has no coverage policy", pkg.ImportPath)
			report.Skipped++
			report.Results = append(report.Results, result)
			continue
		}

		ruleCopy := rule.rule
		result.Rule = &ruleCopy
		item := stats[pkg.ImportPath]
		result.Covered = item.covered
		result.Statements = item.statements
		report.Checked++

		if len(pkg.Files) == 0 {
			result.Skipped = true
			result.Message = fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath)
			report.Checked--
			report.Skipped++
			report.Results = append(report.Results, result)
			continue
		}

		if item.blocks == 0 {
			result.Skipped = true
			result.Message = fmt.Sprintf("package %q has no coverage profile blocks", pkg.ImportPath)
			report.Checked--
			report.Skipped++
			report.Results = append(report.Results, result)
			continue
		}

		if item.statements == 0 {
			result.Skipped = true
			result.Message = fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath)
			report.Checked--
			report.Skipped++
			report.Results = append(report.Results, result)
			continue
		}

		result.Coverage = float64(item.covered) * 100 / float64(item.statements)
		if result.Coverage+1e-9 < rule.rule.Min {
			result.Violation = true
			result.Message = fmt.Sprintf(
				"coverage %.2f%% is below %.2f%% for package %q (%d/%d statements)",
				result.Coverage,
				rule.rule.Min,
				pkg.ImportPath,
				result.Covered,
				result.Statements,
			)
			report.Failed++
		} else {
			result.Message = fmt.Sprintf(
				"coverage %.2f%% meets %.2f%% for package %q (%d/%d statements)",
				result.Coverage,
				rule.rule.Min,
				pkg.ImportPath,
				result.Covered,
				result.Statements,
			)
		}
		report.Results = append(report.Results, result)
	}

	return report
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
		key := blockKey{importPath: match.importPath, file: match.file, position: block.Position}
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
			match := fileMatch{importPath: pkg.ImportPath, file: absolute}
			index.files[absolute] = match

			if rel, err := filepath.Rel(cwd, filepath.FromSlash(absolute)); err == nil && isLocalRelative(rel) {
				index.files[normalizePath(rel)] = match
			}
			if pkg.Dir != "" {
				if rel, err := filepath.Rel(pkg.Dir, filepath.FromSlash(absolute)); err == nil && isLocalRelative(rel) {
					index.files[normalizePath(path.Join(pkg.ImportPath, filepath.ToSlash(rel)))] = match
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

func ValidateMinimum(value float64) error {
	return validateMinimum(value)
}

func validateMinimum(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 100 {
		return fmt.Errorf("min must be finite, greater than 0, and at most 100, got %.2f", value)
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
	return value != "." && value != "" && value != ".." && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}

func normalizePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))
	return strings.TrimPrefix(value, "./")
}
