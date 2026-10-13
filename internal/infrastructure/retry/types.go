// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package retry bounds retries of operations that are safe to repeat.
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
