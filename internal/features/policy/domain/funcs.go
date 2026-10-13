// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	collectiondomain "github.com/gostafa/coverlint/internal/features/collection/domain"
)

// AppendTestFailures adds test-failure violations from coverage metadata.
func AppendTestFailures(
	report *Report,
	packages []collectiondomain.Package,
	coverage *collectiondomain.Coverage,
) {
	if skipTestFailures(report, coverage) {
		return
	}

	appendTestFailureResults(report, packages, coverage)
}

// Evaluate applies the policy to packages and coverage blocks.
func Evaluate(
	policy *Policy,
	packages []collectiondomain.Package,
	blocks []collectiondomain.Block,
) Report {
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

// NewPolicy compiles ordered coverage rules.
func NewPolicy(rules []Rule) (Policy, error) {
	if len(rules) == zero {
		return Policy{}, errMissingCoverageRule
	}

	compiled, err := compileRules(rules)
	if err != nil {
		return Policy{}, fmt.Errorf("compile coverage rules: %w", err)
	}

	return Policy{rules: compiled}, nil
}

// ValidateMinimum checks whether a coverage minimum is allowed.
func ValidateMinimum(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, zero) || value < zero || value > one {
		return fmt.Errorf("%w, got %.2f", errInvalidMinimum, value)
	}

	return nil
}

func addImportPathIndexFile(
	match *fileMatch,
	pkg *collectiondomain.Package,
	extra *importPathExtra,
) {
	if pkg.Dir == emptyString {
		return
	}

	relative, err := filepath.Rel(pkg.Dir, filepath.FromSlash(extra.absolute))

	if err == nil && isLocalRelative(relative) {
		extra.index.files[normalizePath(path.Join(pkg.ImportPath, filepath.ToSlash(relative)))] = *match
	}
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

func aggregate(
	packages []collectiondomain.Package,
	blocks []collectiondomain.Block,
) map[string]packageStats {
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

func appendFailedPackageResults(
	report *Report,
	packages []collectiondomain.Package,
	coverage *collectiondomain.Coverage,
) {
	index := packageIndexByImportPath(packages)

	for i := range coverage.FailedPackages {
		result := testFailureResult(coverage.FailedPackages[i], emptyString)

		result.Message += failureDetails(result.ImportPath, coverage)

		appendPackageTestFailure(report, index, &result)
	}
}

func appendPackageTestFailure(
	report *Report,
	index map[string]collectiondomain.Package,
	result *Result,
) {
	pkg, ok := index[result.ImportPath]

	if ok {
		result.File = pkg.FirstFile
	}

	addReport(report, result)

	report.Results = append(report.Results, *result)
}

func appendSyntheticTestFailure(report *Report, coverage *collectiondomain.Coverage) {
	result := testFailureResult(emptyString, emptyString)

	result.Message += failureDetails(emptyString, coverage)
	addReport(report, &result)

	report.Results = append(report.Results, result)
}

func appendTestFailureResults(
	report *Report,
	packages []collectiondomain.Package,
	coverage *collectiondomain.Coverage,
) {
	if len(coverage.FailedPackages) == zero {
		appendSyntheticTestFailure(report, coverage)

		return
	}

	appendFailedPackageResults(report, packages, coverage)
}

func compileGlob(pattern string) (globPattern, error) {
	if pattern == emptyString {
		return globPattern{}, errEmptyPattern
	}

	segments := strings.Split(pattern, pathSeparator)

	for i := range segments {
		err := validateGlobSegment(segments[i])
		if err != nil {
			return globPattern{}, fmt.Errorf("validate glob segment: %w", err)
		}
	}

	return globPattern{segments: segments}, nil
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

func coverageSkipReason(pkg *collectiondomain.Package, item packageStats) string {
	switch {
	case notCoverable(pkg, item):
		return fmt.Sprintf("package %q has no coverable statements", pkg.ImportPath)
	case item.blocks == zero:
		return fmt.Sprintf("package %q has no coverage profile blocks", pkg.ImportPath)
	default:
		return emptyString
	}
}

func evaluatePackage(policy *Policy, pkg *collectiondomain.Package, item packageStats) Result {
	result := newResult(pkg)
	rule := matchRule(policy, pkg.ImportPath)

	if reason := policySkipReason(pkg, rule); reason != emptyString {
		return skippedResult(&result, reason)
	}

	if reason := coverageSkipReason(pkg, item); reason != emptyString {
		return skippedResult(&result, reason)
	}

	return fillCoverageResult(&result, item, rule)
}

func failureDetails(importPath string, coverage *collectiondomain.Coverage) string {
	if len(coverage.Failures) == zero {
		return failureOutput(coverage.TestOutput)
	}

	details := make([]string, zero, len(coverage.Failures))
	hasTests := hasNamedFailures(importPath, coverage.Failures)

	for i := range coverage.Failures {
		failure := &coverage.Failures[i]

		if includeFailure(importPath, failure, hasTests) {
			details = append(details, formatTestFailure(failure))
		}
	}

	return strings.Join(details, emptyString)
}

func failureOutput(output string) string {
	if strings.TrimSpace(output) == emptyString {
		return emptyString
	}

	return "\n" + strings.TrimRight(output, lineEndings)
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

func formatTestFailure(failure *collectiondomain.TestFailure) string {
	name := emptyString

	if failure.Test != emptyString {
		name = "\nfailed test: " + failure.Test
	}

	return name + failureOutput(failure.Output)
}

func globMatch(matcher *globMatcher, patternIndex, pathIndex int) bool {
	current := [indexPairSize]int{patternIndex, pathIndex}

	if matcher.visited[current] {
		return matcher.memo[current]
	}

	matcher.visited[current] = true

	matched := globMatchAt(matcher, patternIndex, pathIndex)

	matcher.memo[current] = matched

	return matched
}

func globMatchAt(matcher *globMatcher, patternIndex, pathIndex int) bool {
	switch {
	case patternIndex == len(matcher.segments):
		return pathIndex == len(matcher.pathSegments)
	case matcher.segments[patternIndex] == doubleStar:
		return globMatchDoubleStar(matcher, patternIndex, pathIndex)
	case pathIndex < len(matcher.pathSegments):
		return globMatchSegment(matcher, patternIndex, pathIndex)
	default:
		return false
	}
}

func globMatchDoubleStar(matcher *globMatcher, patternIndex, pathIndex int) bool {
	if globMatch(matcher, patternIndex+one, pathIndex) {
		return true
	}

	return pathIndex < len(matcher.pathSegments) && globMatch(matcher, patternIndex, pathIndex+one)
}

func globMatchSegment(matcher *globMatcher, patternIndex, pathIndex int) bool {
	segmentMatched, err := path.Match(
		matcher.segments[patternIndex],
		matcher.pathSegments[pathIndex],
	)
	if err != nil {
		return false
	}

	return segmentMatched && globMatch(matcher, patternIndex+one, pathIndex+one)
}

func globSegmentShape(segment string) (bool, error) {
	switch {
	case segment == emptyString:
		return false, errEmptyPathSegment
	case segment == doubleStar:
		return true, nil
	case strings.Contains(segment, doubleStar):
		return false, errPartialDoubleStar
	default:
		return false, nil
	}
}

func hasNamedFailures(importPath string, failures []collectiondomain.TestFailure) bool {
	for i := range failures {
		if failures[i].ImportPath == importPath && failures[i].Test != emptyString {
			return true
		}
	}

	return false
}

func hasTestFailures(coverage *collectiondomain.Coverage) bool {
	return coverage.TestsFailed || len(coverage.FailedPackages) > zero
}

func includeFailure(importPath string, failure *collectiondomain.TestFailure, hasTests bool) bool {
	if failure.ImportPath == emptyString {
		return true
	}

	return failure.ImportPath == importPath && (failure.Test != emptyString || !hasTests)
}

func indexPackageFiles(index *packageIndex, cwd string, pkg *collectiondomain.Package) {
	for j := range pkg.Files {
		addIndexFile(&indexFileAdd{
			index:    *index,
			cwd:      cwd,
			pkg:      pkg,
			filename: pkg.Files[j],
		})
	}
}

func indexPackages(packages []collectiondomain.Package, cwd string) packageIndex {
	index := packageIndex{files: make(map[string]fileMatch)}

	for i := range packages {
		indexPackageFiles(&index, cwd, &packages[i])
	}

	return index
}

func isLocalRelative(value string) bool {
	return value != currentDir && value != emptyString && value != parentDir &&
		!strings.HasPrefix(value, parentDir+string(filepath.Separator))
}

func lookupIndex(index packageIndex, filename string) fileMatch {
	return index.files[normalizePath(filename)]
}

func matchGlob(pattern globPattern, importPath string) bool {
	return globMatch(newGlobMatcher(pattern, importPath), zero, zero)
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

func mergeBlock(
	merged map[blockKey]mergedBlock,
	index packageIndex,
	block *collectiondomain.Block,
) {
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

func newGlobMatcher(pattern globPattern, importPath string) *globMatcher {
	return &globMatcher{
		segments:     pattern.segments,
		pathSegments: strings.Split(importPath, pathSeparator),
		memo:         make(map[[indexPairSize]int]bool),
		visited:      make(map[[indexPairSize]int]bool),
	}
}

func newPackageIndex(packages []collectiondomain.Package) packageIndex {
	return newPackageIndexWith(packages, os.Getwd)
}

func newPackageIndexWith(
	packages []collectiondomain.Package,
	getwd func() (string, error),
) packageIndex {
	cwd, err := getwd()
	if err != nil {
		cwd = emptyString
	}

	return indexPackages(packages, cwd)
}

func newResult(pkg *collectiondomain.Package) Result {
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

func notCoverable(pkg *collectiondomain.Package, item packageStats) bool {
	return len(pkg.Files) == zero || item.statements == zero
}

func packageFilePath(pkg *collectiondomain.Package, filename string) string {
	if filepath.IsAbs(filename) || pkg.Dir == emptyString {
		return normalizePath(filename)
	}

	return normalizePath(filepath.Join(pkg.Dir, filename))
}

func packageIndexByImportPath(
	packages []collectiondomain.Package,
) map[string]collectiondomain.Package {
	index := make(map[string]collectiondomain.Package, len(packages))

	for i := range packages {
		index[packages[i].ImportPath] = packages[i]
	}

	return index
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

func policySkipReason(pkg *collectiondomain.Package, rule *compiledRule) string {
	if rule == nil {
		return fmt.Sprintf("package %q has no coverage policy", pkg.ImportPath)
	}

	return emptyString
}

func skipTestFailures(report *Report, coverage *collectiondomain.Coverage) bool {
	return report == nil || coverage == nil || !hasTestFailures(coverage)
}

func skippedResult(result *Result, message string) Result {
	result.Skipped = true
	result.Message = message

	return *result
}

func testFailureMessage(importPath string) string {
	if importPath == emptyString {
		return "tests failed"
	}

	return fmt.Sprintf("tests failed for package %q", importPath)
}

func testFailureResult(importPath, file string) Result {
	return Result{
		ImportPath: importPath,
		File:       file,
		Rule:       nil,
		Coverage:   zero,
		Statements: zero,
		Covered:    zero,
		Skipped:    false,
		Violation:  true,
		Message:    testFailureMessage(importPath),
	}
}

func validateGlobSegment(segment string) error {
	skip, err := globSegmentShape(segment)
	if err != nil {
		return fmt.Errorf("validate glob segment shape: %w", err)
	}

	if skip {
		return nil
	}

	err = validateGlobSegmentSyntax(segment)
	if err != nil {
		return fmt.Errorf("validate glob segment syntax: %w", err)
	}

	return nil
}

func validateGlobSegmentSyntax(segment string) error {
	matched, err := path.Match(segment, emptyString)
	if err != nil {
		return fmt.Errorf("compile glob segment: %w", err)
	}

	if matched {
		return nil
	}

	return nil
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
