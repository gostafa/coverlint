// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
)

// AppendTestFailures adds test-failure violations from coverage metadata.
func AppendTestFailures(report *Report, packages []Package, coverage *Coverage) {
	if report == nil || coverage == nil || !hasTestFailures(coverage) {
		return
	}

	if len(coverage.FailedPackages) == zero {
		appendSyntheticTestFailure(report)

		return
	}

	index := packageIndexByImportPath(packages)

	for i := range coverage.FailedPackages {
		appendPackageTestFailure(report, index, coverage.FailedPackages[i])
	}
}

func appendPackageTestFailure(report *Report, index map[string]Package, importPath string) {
	pkg, ok := index[importPath]
	file := emptyString

	if ok {
		file = pkg.FirstFile
	}

	result := testFailureResult(importPath, file)
	addReport(report, &result)

	report.Results = append(report.Results, result)
}

func appendSyntheticTestFailure(report *Report) {
	result := testFailureResult(emptyString, emptyString)
	addReport(report, &result)

	report.Results = append(report.Results, result)
}

func hasTestFailures(coverage *Coverage) bool {
	return len(coverage.FailedPackages) > zero || coverage.TestOutput != emptyString
}

func packageIndexByImportPath(packages []Package) map[string]Package {
	index := make(map[string]Package, len(packages))

	for i := range packages {
		index[packages[i].ImportPath] = packages[i]
	}

	return index
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
