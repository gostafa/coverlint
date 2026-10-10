// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"slices"
	"strings"
)

// MergeTestFailures combines failure metadata without changing current-run output or coverage.
func MergeTestFailures(current, imported *Coverage) {
	current.TestsFailed = current.TestsFailed || imported.TestsFailed

	for i := range imported.FailedPackages {
		pkg := imported.FailedPackages[i]

		if !slices.Contains(current.FailedPackages, pkg) {
			current.FailedPackages = append(current.FailedPackages, pkg)
		}
	}

	current.Failures = mergedFailures(current.Failures, imported.Failures)
}

func mergedFailures(current, imported []TestFailure) []TestFailure {
	result := make([]TestFailure, zero, len(current)+len(imported))

	for i := range current {
		result = mergeFailure(result, &current[i])
	}

	for i := range imported {
		result = mergeFailure(result, &imported[i])
	}

	return result
}

func mergeFailure(result []TestFailure, failure *TestFailure) []TestFailure {
	for i := range result {
		if result[i].ImportPath == failure.ImportPath && result[i].Test == failure.Test {
			result[i].Output = mergeFailureOutput(result[i].Output, failure.Output)

			return result
		}
	}

	return append(result, *failure)
}

func mergeFailureOutput(current, imported string) string {
	if imported == emptyString || containsFailureOutput(current, imported) {
		return current
	}

	if current == emptyString {
		return imported
	}

	return strings.TrimRight(current, lineEndings) + "\n" + imported
}

func containsFailureOutput(current, imported string) bool {
	return strings.Contains("\n"+strings.TrimRight(current, lineEndings)+"\n",
		"\n"+strings.TrimRight(imported, lineEndings)+"\n")
}
