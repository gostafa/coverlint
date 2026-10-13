// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package analyzer provides the go/analysis adapter for coverlint.
package analyzer

import (
	"context"
	"sync"

	"github.com/gostafa/coverlint/coverlint"
)

type (
	// Settings configures the golangci-lint / go/analysis adapter.
	Settings = struct {
		Timeout            string           `json:"timeout,omitempty"`
		TestResultPath     string           `json:"test_result_path,omitempty"`
		CoverageResultPath string           `json:"coverage_result_path,omitempty"`
		Rules              []coverlint.Rule `json:"rules,omitempty"`
		Packages           []string         `json:"packages,omitempty"`
		TestArgs           []string         `json:"test_args,omitempty"`
	}

	runner = struct {
		load     func(context.Context) (map[string]coverlint.Result, error)
		reported sync.Map
	}

	runResult struct{}

	analyzerResult interface {
		isAnalyzerResult()
	}
)
