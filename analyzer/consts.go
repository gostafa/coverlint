// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

const (
	// Name is the analyzer name registered with analysis and golangci-lint.
	Name = "coverlint"

	// Doc is the short analyzer documentation shown by go/analysis tools.
	Doc = "enforce minimum Go test coverage"

	emptyString           = ""
	errUnmarshalSettings  = "UnmarshalSettings: %w"
	errRun                = "run: %w"
	errLoad               = "load: %w"
	errRemapKebabKeys     = "remapKebabKeys: %w"
	camelTestArgsKeyName  = "testArgs"
	legacyTestArgsKeyName = "test-args"
	testArgsKeyName       = "test_args"
)
