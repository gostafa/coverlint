// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package environment

import (
	"os"
)

// WorkingDirectory returns the current directory, or an empty fallback if unavailable.
func WorkingDirectory() string { return workingDirectoryWith(os.Getwd) }

func workingDirectoryWith(getwd func() (string, error)) string {
	value, err := getwd()
	if err != nil {
		return ""
	}

	return value
}
