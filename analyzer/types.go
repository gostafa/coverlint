// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"sync"

	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

type (
	// Settings configures the golangci-lint / go/analysis adapter.
	Settings = struct {
		Timeout  string        `json:"timeout,omitempty"`
		Rules    []domain.Rule `json:"rules,omitempty"`
		Packages []string      `json:"packages,omitempty"`
		TestArgs []string      `json:"test_args,omitempty"`
	}

	runner = struct {
		loadErr    error
		violations map[string]coverlint.Result
		config     *coverlint.Config
		reported   sync.Map
		loadOnce   sync.Once
	}

	runResult struct{}

	analyzerResult interface {
		isAnalyzerResult()
	}
)
