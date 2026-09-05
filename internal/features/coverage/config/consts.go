// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"time"
)

const (
	// DefaultMinimum is the default required package coverage fraction.
	DefaultMinimum             = 0.80
	defaultTimeout             = 10 * time.Minute
	emptyString                = ""
	zero                       = 0
	one                        = 1
	testArgsAliasCount         = 3
	doubleStar                 = "**"
	defaultPattern             = "./..."
	flagPrefix                 = "-"
	doubleDash                 = "--"
	equalsSign                 = "="
	timeoutKey                 = "timeout"
	testArgsKey                = "test_args"
	testArgsCamel              = "testArgs"
	testArgsLegacy             = "test-args"
	testResultPathKey          = "test_result_path"
	testResultPathLegacy       = "test-result-path"
	coverageResultPathKey      = "coverage_result_path"
	coverageResultPathLegacy   = "coverage-result-path"
	wrappedQuotedErr           = "%w: %q"
	errUnmarshalCoverageConfig = "unmarshal coverage config: %w"
	errDecodeCoverageConfig    = "decode coverage config: %w"
	errRemapTestArgsKeys       = "remap test args keys: %w"
	errRemapResultPathKeys     = "remap result path keys: %w"
	errResolveCoverageSettings = "resolve coverage settings: %w"
)
