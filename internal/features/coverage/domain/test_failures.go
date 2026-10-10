// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"strings"
)

// AppendTestFailures adds test-failure violations from coverage metadata.
func AppendTestFailures(report *Report, packages []Package, coverage *Coverage) {
	if skipTestFailures(report, coverage) {
		return
	}

	appendTestFailureResults(report, packages, coverage)
}

func appendFailedPackageResults(report *Report, packages []Package, coverage *Coverage) {
	index := packageIndexByImportPath(packages)

	for i := range coverage.FailedPackages {
		result := testFailureResult(coverage.FailedPackages[i], emptyString)

		result.Message += failureDetails(result.ImportPath, coverage)

		appendPackageTestFailure(report, index, &result)
	}
}

func appendPackageTestFailure(report *Report, index map[string]Package, result *Result) {
	pkg, ok := index[result.ImportPath]

	if ok {
		result.File = pkg.FirstFile
	}

	addReport(report, result)

	report.Results = append(report.Results, *result)
}

func appendSyntheticTestFailure(report *Report, coverage *Coverage) {
	result := testFailureResult(emptyString, emptyString)

	result.Message += failureDetails(emptyString, coverage)
	addReport(report, &result)

	report.Results = append(report.Results, result)
}

func appendTestFailureResults(report *Report, packages []Package, coverage *Coverage) {
	if len(coverage.FailedPackages) == zero {
		appendSyntheticTestFailure(report, coverage)

		return
	}

	appendFailedPackageResults(report, packages, coverage)
}

func failureDetails(importPath string, coverage *Coverage) string {
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

func formatTestFailure(failure *TestFailure) string {
	name := emptyString

	if failure.Test != emptyString {
		name = "\nfailed test: " + failure.Test
	}

	return name + failureOutput(failure.Output)
}

func hasNamedFailures(importPath string, failures []TestFailure) bool {
	for i := range failures {
		if failures[i].ImportPath == importPath && failures[i].Test != emptyString {
			return true
		}
	}

	return false
}

func includeFailure(importPath string, failure *TestFailure, hasTests bool) bool {
	if failure.ImportPath == emptyString {
		return true
	}

	return failure.ImportPath == importPath && (failure.Test != emptyString || !hasTests)
}

func hasTestFailures(coverage *Coverage) bool {
	return coverage.TestsFailed || len(coverage.FailedPackages) > zero
}

func packageIndexByImportPath(packages []Package) map[string]Package {
	index := make(map[string]Package, len(packages))

	for i := range packages {
		index[packages[i].ImportPath] = packages[i]
	}

	return index
}

func skipTestFailures(report *Report, coverage *Coverage) bool {
	return report == nil || coverage == nil || !hasTestFailures(coverage)
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
