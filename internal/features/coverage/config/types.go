// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

type (
	// JSONObject decodes coverage settings from JSON.
	jsonObject interface {
		decode(data []byte) error
	}

	decoderFunc func([]byte) error

	// Config contains user-provided coverage settings.
	Config = struct {
		Timeout            string        `json:"timeout,omitempty"`
		TestResultPath     string        `json:"test_result_path,omitempty"`
		CoverageResultPath string        `json:"coverage_result_path,omitempty"`
		Rules              []domain.Rule `json:"rules,omitempty"`
		Packages           []string      `json:"packages,omitempty"`
		TestArgs           []string      `json:"test_args,omitempty"`
	}

	// Resolved contains validated settings ready for coverage execution.
	Resolved = struct {
		TestResultPath     string
		CoverageResultPath string
		Policy             domain.Policy
		Patterns           []string
		TestArgs           []string
		Timeout            time.Duration
	}

	resolvedBuild = struct {
		input           *Config
		policy          *domain.Policy
		packagePatterns []string
		testArgs        []string
		timeout         time.Duration
	}
)
