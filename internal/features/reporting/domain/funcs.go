// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"path/filepath"
)

// Diagnostic formats a coverage result as a linter diagnostic line.
func Diagnostic(result *Result, linterName, cwd string) string {
	location := diagnosticLocation(result, cwd)

	return fmt.Sprintf(diagPrefix, location, result.Message, linterName)
}

func diagnosticLocation(result *Result, cwd string) string {
	if result.File == emptyString {
		return filepath.ToSlash(result.ImportPath)
	}

	return filepath.ToSlash(relativeLocation(result.File, cwd))
}

func relativeLocation(location, cwd string) string {
	if cwd == emptyString {
		return location
	}

	relative, err := filepath.Rel(cwd, location)

	if err != nil || relative == emptyString {
		return location
	}

	return relative
}
