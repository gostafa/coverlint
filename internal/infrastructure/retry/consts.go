// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package retry

import (
	"time"
)

const (
	zero = 0

	one = 1

	maxAttempts = 3

	initialInterval = 100 * time.Millisecond

	maximumInterval = 500 * time.Millisecond

	elapsedBudget = 2 * time.Second

	multiplier = 2

	jitter = 0.2

	jitterRange = 2 * jitter

	jitterDigits = 10

	jitterAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
)
