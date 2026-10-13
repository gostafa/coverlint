// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package analyzer provides the go/analysis adapter for coverlint.
package analyzer

import (
	"context"
	"sync"

	"github.com/gostafa/coverlint/coverlint"
	reportingports "github.com/gostafa/coverlint/internal/features/policy/ports/resultcache"
)

type (
	// snapshotProvider supplies immutable diagnostics through the owned cache port.
	snapshotProvider struct {
		get func(context.Context) (map[string]reportingports.Violation, error)
	}

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

	runner = struct {
		load     reportingports.ResultProvider
		reported sync.Map
	}

	runResult struct{}

	analyzerResult interface {
		isAnalyzerResult()
	}
)
