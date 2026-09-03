// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"errors"
)

var (
	errMissingCoverageRule = errors.New("at least one coverage rule is required")
	errMissingRulePattern  = errors.New("pattern is required")
	errInvalidMinimum      = errors.New("min must be finite and in [0, 1]")
	errEmptyPattern        = errors.New("pattern is empty")
	errEmptyPathSegment    = errors.New("pattern contains an empty path segment")
	errPartialDoubleStar   = errors.New("** must be a complete path segment")

	_ pathMatcher   = Rule{}
	_ coverageGate  = Rule{}
	_ patternSource = Rule{}
)
