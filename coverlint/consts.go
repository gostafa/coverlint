// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"github.com/gostafa/coverlint/internal/features/coverage/config"
)

const (
	// Name is the golangci-lint plugin and diagnostic category name.
	Name = "coverlint"
	// DefaultMinimum is the default required package coverage fraction.
	DefaultMinimum    = config.DefaultMinimum
	zero              = 0
	emptyString       = ""
	currentDir        = "."
	resultDirPerm     = 0o700
	resultFilePerm    = 0o600
	errCheckWrap      = "Check: %w"
	errOpenHTMLReport = "open HTML coverage report: %w"
)
