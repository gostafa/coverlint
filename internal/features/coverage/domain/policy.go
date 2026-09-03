// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"
)

// NewPolicy compiles ordered coverage rules and exclude patterns.
func NewPolicy(rules []Rule, excludes []string) (Policy, error) {
	if len(rules) == zero {
		return Policy{}, errMissingCoverageRule
	}

	compiled, err := compileRules(rules)
	if err != nil {
		return Policy{}, fmt.Errorf("compile coverage rules: %w", err)
	}

	compiledExcludes, err := compileExcludes(excludes)
	if err != nil {
		return Policy{}, fmt.Errorf("compile coverage excludes: %w", err)
	}

	return Policy{rules: compiled, excludes: compiledExcludes}, nil
}

// Matches reports whether the rule pattern matches importPath.
func (rule Rule) Matches(importPath string) bool {
	glob, err := compileGlob(rule.Pattern)
	if err != nil {
		return false
	}

	return matchGlob(glob, importPath)
}

// MinCoverage returns the minimum acceptable coverage fraction for this rule.
func (rule Rule) MinCoverage() float64 {
	return rule.Min
}

// PatternValue returns the glob pattern for this rule.
func (rule Rule) PatternValue() string {
	return rule.Pattern
}

// String returns the pattern and minimum as "pattern:min".
func (rule Rule) String() string {
	return fmt.Sprintf("%s:%.2f", rule.Pattern, rule.Min)
}

func addIndexFile(input *indexFileAdd) {
	absolute := packageFilePath(input.pkg, input.filename)
	match := fileMatch{importPath: input.pkg.ImportPath, file: absolute}

	input.index.files[absolute] = match
	addRelativeIndexFile(input.index, &match, &relativePaths{
		cwd:      input.cwd,
		absolute: absolute,
	})
	addImportPathIndexFile(&match, input.pkg, &importPathExtra{
		index:    input.index,
		absolute: absolute,
	})
}

func addImportPathIndexFile(match *fileMatch, pkg *Package, extra *importPathExtra) {
	if pkg.Dir == emptyString {
		return
	}

	relative, err := filepath.Rel(pkg.Dir, filepath.FromSlash(extra.absolute))

	if err == nil && isLocalRelative(relative) {
		extra.index.files[normalizePath(path.Join(pkg.ImportPath, filepath.ToSlash(relative)))] = *match
	}
}

func addMergedBlock(stats map[string]packageStats, key *blockKey, block *mergedBlock) {
	if block.statements == zero {
		return
	}

	importPath := key[zero]
	item := stats[importPath]
	item.blocks++

	item.statements += block.statements

	if block.covered {
		item.covered += block.statements
	}

	stats[importPath] = item
}

func addRelativeIndexFile(index packageIndex, match *fileMatch, paths *relativePaths) {
	relative, err := filepath.Rel(paths.cwd, filepath.FromSlash(paths.absolute))

	if err == nil && isLocalRelative(relative) {
		index.files[normalizePath(relative)] = *match
	}
}

func addReport(report *Report, result *Result) {
	switch {
	case result.Skipped:
		report.Skipped++
	case result.Violation:
		report.Checked++

		report.Failed++
	default:
		report.Checked++
	}
}

func aggregate(packages []Package, blocks []Block) map[string]packageStats {
	index := newPackageIndex(packages)
	stats := make(map[string]packageStats, len(packages))
	merged := make(map[blockKey]mergedBlock, len(blocks))

	for i := range blocks {
		mergeBlock(merged, index, &blocks[i])
	}

	for key := range merged {
		block := merged[key]
		addMergedBlock(stats, &key, &block)
	}

	return stats
}

func compileExcludes(excludes []string) ([]globPattern, error) {
	compiled := make([]globPattern, zero, len(excludes))

	for index := range excludes {
		glob, err := compileGlob(excludes[index])
		if err != nil {
			return nil, fmt.Errorf(
				"exclude %d: invalid glob %q: %w",
				index+one,
				excludes[index],
				err,
			)
		}

		compiled = append(compiled, glob)
	}

	return compiled, nil
}

func compileRule(index int, rule *Rule) (compiledRule, error) {
	if rule.Pattern == emptyString {
		return compiledRule{}, fmt.Errorf(ruleErrorFormat, index, errMissingRulePattern)
	}

	err := ValidateMinimum(rule.Min)
	if err != nil {
		return compiledRule{}, fmt.Errorf(ruleErrorFormat, index, err)
	}

	glob, err := compileGlob(rule.Pattern)
	if err != nil {
		return compiledRule{}, fmt.Errorf(
			"rule %d: invalid glob %q: %w",
			index,
			rule.Pattern,
			err,
		)
	}

	return compiledRule{rule: *rule, glob: glob}, nil
}

func compileRules(rules []Rule) ([]compiledRule, error) {
	compiled := make([]compiledRule, zero, len(rules))

	for index := range rules {
		item, err := compileRule(index+one, &rules[index])
		if err != nil {
			return nil, fmt.Errorf("compile coverage rule: %w", err)
		}

		compiled = append(compiled, item)
	}

	return compiled, nil
}

func coverageMessage(result *Result, minimum float64) string {
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

func coverageSkipReason(pkg *Package, item packageStats) string {
	switch {
	case notCoverable(pkg, item):
		return fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath)
	case item.blocks == zero:
		return fmt.Sprintf("package %q has no coverage profile blocks", pkg.ImportPath)
	default:
		return emptyString
	}
}

// Evaluate applies the policy to packages and coverage blocks.
func Evaluate(policy *Policy, packages []Package, blocks []Block) Report {
	stats := aggregate(packages, blocks)
	report := Report{
		Results: make([]Result, zero, len(packages)),
		Checked: zero,
		Failed:  zero,
		Skipped: zero,
	}

	for i := range packages {
		result := evaluatePackage(policy, &packages[i], stats[packages[i].ImportPath])
		addReport(&report, &result)

		report.Results = append(report.Results, result)
	}

	return report
}

func evaluatePackage(policy *Policy, pkg *Package, item packageStats) Result {
	result := newResult(pkg)
	rule := matchRule(policy, pkg.ImportPath)

	if reason := policySkipReason(policy, pkg, rule); reason != emptyString {
		return skippedResult(&result, reason)
	}

	if reason := coverageSkipReason(pkg, item); reason != emptyString {
		return skippedResult(&result, reason)
	}

	return fillCoverageResult(&result, item, rule)
}

func excluded(policy *Policy, importPath string) bool {
	for i := range policy.excludes {
		if matchGlob(policy.excludes[i], importPath) {
			return true
		}
	}

	return false
}

func fillCoverageResult(result *Result, item packageStats, rule *compiledRule) Result {
	ruleCopy := rule.rule

	result.Rule = &ruleCopy
	result.Covered = item.covered
	result.Statements = item.statements
	result.Coverage = float64(item.covered) * percentageMultiplier / float64(item.statements)

	threshold := rule.rule.MinCoverage() * percentageMultiplier

	result.Violation = result.Coverage+coverageComparisonEpsilon < threshold
	result.Message = coverageMessage(result, threshold)

	return *result
}

func isLocalRelative(value string) bool {
	return value != currentDir && value != emptyString && value != parentDir &&
		!strings.HasPrefix(value, parentDir+string(filepath.Separator))
}

func lookupIndex(index packageIndex, filename string) fileMatch {
	return index.files[normalizePath(filename)]
}

func matchRule(policy *Policy, importPath string) *compiledRule {
	var selected *compiledRule

	for i := range policy.rules {
		rule := matchingCandidate(&policy.rules[i], importPath, selected)

		if rule == nil {
			continue
		}

		selected = rule
	}

	return selected
}

func matchingCandidate(rule *compiledRule, importPath string, current *compiledRule) *compiledRule {
	if !matchGlob(rule.glob, importPath) {
		return nil
	}

	if current != nil && !moreSpecific(rule.rule.PatternValue(), current.rule.PatternValue()) {
		return nil
	}

	return rule
}

func mergeBlock(merged map[blockKey]mergedBlock, index packageIndex, block *Block) {
	match := lookupIndex(index, block.File)

	if match.importPath == emptyString {
		return
	}

	key := blockKey{
		match.importPath,
		match.file + blockIdentitySeparator + block.Position,
	}
	item := merged[key]

	item.statements = block.Statements
	item.covered = item.covered || block.Covered
	merged[key] = item
}

func moreSpecific(candidate, current string) bool {
	candidateLiteral, candidateWildcards, candidateSegments := patternSpecificity(candidate)
	currentLiteral, currentWildcards, currentSegments := patternSpecificity(current)

	if candidateLiteral != currentLiteral {
		return candidateLiteral > currentLiteral
	}

	if candidateWildcards != currentWildcards {
		return candidateWildcards < currentWildcards
	}

	return candidateSegments >= currentSegments
}

func newPackageIndex(packages []Package) packageIndex {
	cwd, err := getwd()
	if err != nil {
		cwd = emptyString
	}

	index := packageIndex{files: make(map[string]fileMatch)}

	for i := range packages {
		for j := range packages[i].Files {
			addIndexFile(&indexFileAdd{
				index:    index,
				cwd:      cwd,
				pkg:      &packages[i],
				filename: packages[i].Files[j],
			})
		}
	}

	return index
}

func newResult(pkg *Package) Result {
	return Result{
		ImportPath: pkg.ImportPath,
		File:       pkg.FirstFile,
		Rule:       nil,
		Coverage:   zero,
		Statements: zero,
		Covered:    zero,
		Skipped:    false,
		Violation:  false,
		Message:    emptyString,
	}
}

func normalizePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))

	return strings.TrimPrefix(value, relativePrefix)
}

func notCoverable(pkg *Package, item packageStats) bool {
	return len(pkg.Files) == zero || item.statements == zero
}

func packageFilePath(pkg *Package, filename string) string {
	if filepath.IsAbs(filename) || pkg.Dir == emptyString {
		return normalizePath(filename)
	}

	return normalizePath(filepath.Join(pkg.Dir, filename))
}

func patternSpecificity(pattern string) (literal, wildcards, segments int) {
	parts := strings.Split(pattern, pathSeparator)

	for i := range parts {
		segments++

		if parts[i] == singleStar || parts[i] == doubleStar {
			wildcards++

			continue
		}

		literal++
	}

	return literal, wildcards, segments
}

func policySkipReason(policy *Policy, pkg *Package, rule *compiledRule) string {
	if excluded(policy, pkg.ImportPath) {
		return fmt.Sprintf("package %q is excluded", pkg.ImportPath)
	}

	if rule == nil {
		return fmt.Sprintf("package %q has no coverage policy", pkg.ImportPath)
	}

	return emptyString
}

func skippedResult(result *Result, message string) Result {
	result.Skipped = true
	result.Message = message

	return *result
}

// ValidateMinimum checks whether a coverage minimum is allowed.
func ValidateMinimum(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, zero) || value < zero || value > one {
		return fmt.Errorf("%w, got %.2f", errInvalidMinimum, value)
	}

	return nil
}
