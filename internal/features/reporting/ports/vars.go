// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"errors"
)

// ErrReporting identifies HTML rendering failures.
var ErrReporting = errors.New("coverage reporting failed")
