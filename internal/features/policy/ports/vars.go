// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"errors"
)

var (
	// ErrCollection identifies input collection or imported-report failures.
	ErrCollection = errors.New("coverage collection failed")

	// ErrResultWrite identifies artifact persistence failures.
	ErrResultWrite = errors.New("coverage result write failed")
)
