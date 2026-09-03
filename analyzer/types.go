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
		Min       float64       `json:"min,omitempty"`
		Overrides []domain.Rule `json:"overrides,omitempty"`
		Exclude   []string      `json:"exclude,omitempty"`
		Packages  []string      `json:"packages,omitempty"`
		Timeout   string        `json:"timeout,omitempty"`
		TestArgs  []string      `json:"testArgs,omitempty"`
	}

	runner struct {
		loadOnce   sync.Once
		loadErr    error
		violations map[string]coverlint.Result
		reported   sync.Map
		config     coverlint.Config
	}

	runResult struct{}
)
