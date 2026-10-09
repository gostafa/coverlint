// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

type (
	// TestFailure contains captured output for a failed test or package.
	TestFailure = struct {
		ImportPath string
		Test       string
		Output     string
	}
)
