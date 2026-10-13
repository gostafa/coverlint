// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"time"
)

type (
	retryAttempt = struct {
		operation func() error
		started   time.Time
		attempts  int
	}
)
