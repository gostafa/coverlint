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

	compiled, err := compileRules(rules)
	if err != nil {
		return Policy{}, err
	}

	compiledExcludes, err := compileExcludes(excludes)
	if err != nil {
		return Policy{}, err
	}

	return Policy{
		rules:    compiled,
		excludes: compiledExcludes,
	}, nil
}

func compileRules(rules []Rule) ([]compiledRule, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for index, rule := range rules {
		item, err := compileRule(index+1, rule)
		if err != nil {
			return nil, err
		}

		compiled = append(compiled, item)
	}

	return compiled, nil
}

func compileRule(index int, rule Rule) (compiledRule, error) {
	if rule.Pattern == "" {
		return compiledRule{}, fmt.Errorf("override %d: %w", index, errMissingRulePattern)
	}

	err := validateMinimum(rule.Min)
	if err != nil {
		return compiledRule{}, fmt.Errorf("override %d: %w", index, err)
	}

	glob, err := compileGlob(rule.Pattern)
	if err != nil {
		return compiledRule{}, fmt.Errorf(
			"override %d: invalid glob %q: %w",
			index,
			rule.Pattern,
			err,
		)
	}

	return compiledRule{rule: rule, glob: glob}, nil
}

func compileExcludes(excludes []string) ([]globPattern, error) {
	compiled := make([]globPattern, 0, len(excludes))
	for index, pattern := range excludes {
		glob, err := compileGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("exclude %d: invalid glob %q: %w", index+1, pattern, err)
		}

		compiled = append(compiled, glob)
	}

	return compiled, nil
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

	rule := p.match(pkg.ImportPath)
	if reason := p.skipReason(pkg, item, rule); reason != "" {
		return skippedResult(result, reason)
	}

	ruleCopy := rule.rule
	result.Rule = &ruleCopy
	result.Covered = item.covered
	result.Statements = item.statements

	result.Coverage = float64(item.covered) * percentageMultiplier / float64(item.statements)
	result.Violation = result.Coverage+coverageComparisonEpsilon < rule.rule.Min
	result.Message = coverageMessage(result, rule.rule.Min)

	return result
}

func (p Policy) skipReason(pkg Package, item packageStats, rule *compiledRule) string {
	reason := p.policySkipReason(pkg, rule)
	if reason != "" {
		return reason
	}

	return coverageSkipReason(pkg, item)
}

func (p Policy) policySkipReason(pkg Package, rule *compiledRule) string {
	if p.excluded(pkg.ImportPath) {
		return fmt.Sprintf("package %q is excluded", pkg.ImportPath)
	}

	if rule == nil {
		return fmt.Sprintf("package %q has no coverage policy", pkg.ImportPath)
	}

	return ""
}

func coverageSkipReason(pkg Package, item packageStats) string {
	switch {
	case notCoverable(pkg, item):
		return fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath)
	case item.blocks == 0:
		return fmt.Sprintf("package %q has no coverage profile blocks", pkg.ImportPath)
	default:
		return ""
	}
}

func notCoverable(pkg Package, item packageStats) bool {
	return len(pkg.Files) == 0 || item.statements == 0
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
		mergeBlock(merged, index, block)
	}

	for key, block := range merged {
		addMergedBlock(stats, key, block)
	}

	return stats
}

func mergeBlock(merged map[blockKey]mergedBlock, index packageIndex, block Block) {
	match := index.lookup(block.File)
	if match.importPath == "" {
		return
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

func addMergedBlock(stats map[string]packageStats, key blockKey, block mergedBlock) {
	if block.statements == 0 {
		return
	}

	item := stats[key.importPath]
	item.blocks++

	item.statements += block.statements
	if block.covered {
		item.covered += block.statements
	}

	stats[key.importPath] = item
}

func newPackageIndex(packages []Package) packageIndex {
	cwd, _ := os.Getwd()
	index := packageIndex{
		files: make(map[string]fileMatch),
	}

	for _, pkg := range packages {
		for _, filename := range pkg.Files {
			index.addFile(cwd, pkg, filename)
		}
	}

	return index
}

func (i packageIndex) addFile(cwd string, pkg Package, filename string) {
	absolute := packageFilePath(pkg, filename)
	match := fileMatch{
		importPath: pkg.ImportPath,
		file:       absolute,
	}
	i.files[absolute] = match
	i.addRelativeFile(cwd, absolute, match)
	i.addImportPathFile(pkg, absolute, match)
}

func packageFilePath(pkg Package, filename string) string {
	if filepath.IsAbs(filename) || pkg.Dir == "" {
		return normalizePath(filename)
	}

	return normalizePath(filepath.Join(pkg.Dir, filename))
}

func (i packageIndex) addRelativeFile(cwd, absolute string, match fileMatch) {
	relative, err := filepath.Rel(cwd, filepath.FromSlash(absolute))
	if err == nil && isLocalRelative(relative) {
		i.files[normalizePath(relative)] = match
	}
}

func (i packageIndex) addImportPathFile(pkg Package, absolute string, match fileMatch) {
	if pkg.Dir == "" {
		return
	}

	relative, err := filepath.Rel(pkg.Dir, filepath.FromSlash(absolute))
	if err == nil && isLocalRelative(relative) {
		i.files[normalizePath(path.Join(pkg.ImportPath, filepath.ToSlash(relative)))] = match
	}
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
