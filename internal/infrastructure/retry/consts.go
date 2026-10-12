// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"time"
)

const (
	maxAttempts     = 3
	initialInterval = 100 * time.Millisecond
	maximumInterval = 500 * time.Millisecond
	elapsedBudget   = 2 * time.Second
	multiplier      = 2
	jitter          = 0.2
)
