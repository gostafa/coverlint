// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain_test

import (
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func TestAppendTestFailuresAddsPackageViolations(t *testing.T) {
	t.Parallel()

	report := domain.Report{
		Results: nil,
		Checked: 0,
		Failed:  0,
		Skipped: 0,
	}
	packages := []domain.Package{{
		ImportPath: "example.com/pkg",
		Dir:        "/repo/pkg",
		Files:      []string{"/repo/pkg/a.go"},
		FirstFile:  "/repo/pkg/a.go",
	}}
	coverage := domain.Coverage{
		Profile:        nil,
		Blocks:         nil,
		FailedPackages: []string{"example.com/pkg"},
		TestOutput:     "FAIL\texample.com/pkg\t0.01s\n",
		TestsFailed:    true,
	}

	domain.AppendTestFailures(&report, packages, &coverage)

	if report.Failed != 1 || report.Checked != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}

	result := report.Results[0]

	if !result.Violation || result.File != "/repo/pkg/a.go" {
		t.Fatalf("result = %#v", result)
	}

	if !strings.Contains(result.Message, `tests failed for package "example.com/pkg"`) {
		t.Fatalf("Message = %q", result.Message)
	}
}

func TestAppendTestFailuresAddsSyntheticViolation(t *testing.T) {
	t.Parallel()

	report := domain.Report{
		Results: nil,
		Checked: 0,
		Failed:  0,
		Skipped: 0,
	}
	coverage := domain.Coverage{
		Profile:        nil,
		Blocks:         nil,
		FailedPackages: nil,
		TestOutput:     "something broke",
		TestsFailed:    true,
	}

	domain.AppendTestFailures(&report, nil, &coverage)

	if report.Failed != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}

	if report.Results[0].Message != "tests failed" {
		t.Fatalf("Message = %q", report.Results[0].Message)
	}
}

func TestAppendTestFailuresNoopWithoutFailures(t *testing.T) {
	t.Parallel()

	report := domain.Report{
		Results: nil,
		Checked: 1,
		Failed:  0,
		Skipped: 0,
	}

	domain.AppendTestFailures(&report, nil, &domain.Coverage{
		Profile:        nil,
		Blocks:         nil,
		FailedPackages: nil,
		TestOutput:     "ok\texample.com/pkg\t0.01s\n",
		TestsFailed:    false,
	})

	if report.Failed != 0 || len(report.Results) != 0 {
		t.Fatalf("report = %#v", report)
	}
}
