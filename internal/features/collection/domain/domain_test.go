// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"strings"
	"testing"
)

func TestMergeTestFailures(t *testing.T) {
	t.Parallel()
	current := Coverage{
		TestsFailed:    true,
		TestOutput:     "current output",
		FailedPackages: []string{"pkg"},
		Failures: []TestFailure{
			{ImportPath: "pkg", Test: "TestCalc", Output: "current assertion"},
			{ImportPath: "pkg", Test: "TestCalc", Output: "current assertion"},
		},
	}
	imported := Coverage{
		TestsFailed:    true,
		FailedPackages: []string{"pkg", "other"},
		Failures: []TestFailure{
			{ImportPath: "pkg", Test: "TestCalc", Output: "report assertion"},
			{ImportPath: "pkg", Test: "TestCalc", Output: "report assertion"},
			{ImportPath: "other", Output: "package failure"},
		},
	}
	MergeTestFailures(&current, &imported)
	if len(current.Failures) != 2 || len(current.FailedPackages) != 2 ||
		current.TestOutput != "current output" {
		t.Fatalf("coverage=%#v", current)
	}
	if strings.Count(current.Failures[0].Output, "current assertion") != 1 ||
		strings.Count(current.Failures[0].Output, "report assertion") != 1 {
		t.Fatalf("output=%q", current.Failures[0].Output)
	}
}

func TestMergeFailureOutputEmpty(t *testing.T) {
	t.Parallel()
	if output := mergeFailureOutput("current", ""); output != "current" {
		t.Fatalf("output=%q", output)
	}
	if output := mergeFailureOutput("", "imported"); output != "imported" {
		t.Fatalf("output=%q", output)
	}
}
