// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

type (
	// Result supplies only the values used to format a diagnostic.
	Result = struct {
		// ImportPath identifies the Go package.
		ImportPath string
		// File locates the diagnostic or coverage block source.
		File string
		// Message contains the user-facing diagnostic.
		Message string
	}
)
