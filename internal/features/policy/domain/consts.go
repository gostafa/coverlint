// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"time"
)

const (
	percentageMultiplier = 100

	coverageComparisonEpsilon = 1e-9

	emptyString = ""

	lineEndings = "\r\n"

	zero = 0

	one = 1

	indexPairSize = 2

	doubleStar = "**"

	singleStar = "*"

	pathSeparator = "/"

	currentDir = "."

	parentDir = ".."

	relativePrefix = "./"

	ruleErrorFormat = "rule %d: %w"

	blockIdentitySeparator = "\x00"

	// DefaultMinimum is the default coverage fraction.
	DefaultMinimum = 0.80

	// DefaultTimeout is the default execution limit.
	DefaultTimeout = 10 * time.Minute

	defaultPattern = "./..."

	flagPrefix = "-"

	equalsSign = "="

	doubleDash = "--"

	wrappedQuotedErr = "%w: %q"
)
