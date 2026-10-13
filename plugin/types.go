// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package plugin adapts the coverlint analyzer to golangci-lint.
package plugin

import (
	"golang.org/x/tools/go/analysis"
)

type (
	// Plugin is the golangci-lint module plugin for coverlint.
	Plugin func() ([]*analysis.Analyzer, error)

	loadMode struct{}
)
