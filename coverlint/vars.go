// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"errors"
)

var (
	// ErrInvalidConfig identifies invalid coverage configuration.
	ErrInvalidConfig = errors.New("invalid coverage configuration")

	// ErrCollection identifies coverage collection and report-import failures.
	ErrCollection = errors.New("coverage collection failed")

	// ErrResultWrite identifies artifact write failures.
	ErrResultWrite = errors.New("coverage result write failed")

	// ErrReporting identifies HTML report failures.
	ErrReporting = errors.New("coverage reporting failed")

	errHTMLAdapterNotConfigured = errors.New("HTML coverage adapter is not configured")

	_ webOpener = (*Run)(nil)
)
