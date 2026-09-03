// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverage

import (
	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/coverage/config"
)

const (
	// Name is the golangci-lint plugin and diagnostic category name.
	Name = coverlint.Name
	// DefaultMinimum is the default required package coverage fraction.
	DefaultMinimum = config.DefaultMinimum
)
