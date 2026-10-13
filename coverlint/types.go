// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"context"
	"io"

	reportingpublic "github.com/gostafa/coverlint/internal/features/reporting/public"
)

type (
	boundaryError func() boundaryDetails

	boundaryDetails = struct {
		category error
		message  string
		canceled bool
		deadline bool
	}

	htmlReporter = reportingpublic.HTMLReporter

	// Rule is a package coverage rule.
	Rule struct {
		// Pattern matches package import paths.
		Pattern string `json:"pattern"`
		// Min is the minimum required coverage fraction.
		Min float64 `json:"min"`
	}

	// Config contains user-provided coverage settings.
	Config = struct {
		// Timeout bounds coverage execution.
		Timeout string `json:"timeout,omitempty"`
		// TestResultPath selects the imported and rewritten test report.
		TestResultPath string `json:"test_result_path,omitempty"`
		// CoverageResultPath selects the saved coverage profile.
		CoverageResultPath string `json:"coverage_result_path,omitempty"`
		// Rules contains ordered coverage thresholds.
		Rules []Rule `json:"rules,omitempty"`
		// Packages selects package patterns or describes collected package metadata.
		Packages []string `json:"packages,omitempty"`
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string `json:"test_args,omitempty"`
	}

	// Result describes one package's coverage policy outcome.
	Result = struct {
		// Rule identifies the selected coverage rule, or is nil for synthetic outcomes.
		Rule *Rule
		// ImportPath identifies the Go package.
		ImportPath string
		// File locates the diagnostic or coverage block source.
		File string
		// Message contains the user-facing diagnostic.
		Message string
		// Coverage contains coverage data or the computed coverage percentage.
		Coverage float64
		// Statements counts coverable statements.
		Statements int64
		// Covered records covered statements or whether the block executed.
		Covered int64
		// Skipped records packages excluded from evaluation.
		Skipped bool
		// Violation reports a policy or test failure.
		Violation bool
	}

	// Report summarizes coverage policy outcomes.
	Report = struct {
		// Results contains individual package outcomes.
		Results []Result
		// Checked counts evaluated packages.
		Checked int
		// Failed counts reported violations.
		Failed int
		// Skipped records packages excluded from evaluation.
		Skipped int
	}

	webOpener interface {
		OpenWeb(ctx context.Context, stdout, stderr io.Writer) error
	}

	htmlOpenArgs = struct {
		stdout  io.Writer
		stderr  io.Writer
		profile []byte
		checked int
	}

	htmlOpener = func(ctx context.Context, args *htmlOpenArgs) error

	// Run contains a completed coverage check and deferred report actions.
	Run struct {
		html    htmlOpener
		profile []byte
		// Report summarizes coverage policy outcomes.
		Report Report
	}
)
