// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"time"

	"github.com/gostafa/coverlint/internal/features/policy/domain"
)

type (
	// JSONObject decodes coverage settings from JSON.
	jsonObject interface {
		decode(data []byte) error
	}

	decoderFunc func([]byte) error

	// Config contains user-provided coverage settings.
	Config = struct {
		// Timeout bounds coverage execution.
		Timeout string `json:"timeout,omitempty"`
		// TestResultPath selects the imported and rewritten test report.
		TestResultPath string `json:"test_result_path,omitempty"`
		// CoverageResultPath selects the saved coverage profile.
		CoverageResultPath string `json:"coverage_result_path,omitempty"`
		// Rules contains ordered coverage thresholds.
		Rules []domain.Rule `json:"rules,omitempty"`
		// Packages selects package patterns or describes collected package metadata.
		Packages []string `json:"packages,omitempty"`
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string `json:"test_args,omitempty"`
	}

	// Resolved contains validated settings ready for coverage execution.
	Resolved = struct {
		// TestResultPath selects the imported and rewritten test report.
		TestResultPath string
		// CoverageResultPath selects the saved coverage profile.
		CoverageResultPath string
		// WorkingDirectory provides the caller's path-resolution context.
		WorkingDirectory string
		Policy           domain.Policy
		// Patterns selects packages for the operation.
		Patterns []string
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string
		// Timeout bounds coverage execution.
		Timeout time.Duration
	}

	resolvedBuild = struct {
		input           *Config
		policy          *domain.Policy
		packagePatterns []string
		testArgs        []string
		timeout         time.Duration
	}
)
