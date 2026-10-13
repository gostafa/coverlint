// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"github.com/gostafa/coverlint/coverlint"
)

type (
	// Settings configures the golangci-lint / go/analysis adapter.
	Settings = struct {
		// Timeout bounds coverage execution.
		Timeout string `json:"timeout,omitempty"`
		// TestResultPath selects the imported and rewritten test report.
		TestResultPath string `json:"test_result_path,omitempty"`
		// CoverageResultPath selects the saved coverage profile.
		CoverageResultPath string `json:"coverage_result_path,omitempty"`
		// Rules contains ordered coverage thresholds.
		Rules []coverlint.Rule `json:"rules,omitempty"`
		// Packages selects package patterns or describes collected package metadata.
		Packages []string `json:"packages,omitempty"`
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string `json:"test_args,omitempty"`
	}
)
