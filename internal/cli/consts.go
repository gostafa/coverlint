// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"time"
)

const (
	version         = "0.5.0"
	defaultTimeout  = 10 * time.Minute
	usageExitCode   = 2
	successExitCode = 0
	failureExitCode = 1
	emptyString     = ""
	floatBitSize    = 64
	ruleSeparator   = ":"
	examplesHeader  = "Examples:"
	ruleFlag        = "rule"
	timeoutFlag     = "timeout"
	errWriteSummary = "writeSummary: %w"
)
