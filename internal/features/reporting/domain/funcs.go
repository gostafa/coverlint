// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"path/filepath"

	coveragedomain "github.com/gostafa/coverlint/internal/features/coverage/domain"
)

// Diagnostic formats a coverage result as a linter diagnostic line.
func Diagnostic(result *coveragedomain.Result, linterName string) string {
	location := diagnosticLocation(result)

	return fmt.Sprintf(diagPrefix, location, result.Message, linterName)
}

func diagnosticLocation(result *coveragedomain.Result) string {
	if result.File == emptyString {
		return filepath.ToSlash(result.ImportPath)
	}

	return filepath.ToSlash(relativeLocation(result.File))
}

func relativeLocation(location string) string {
	workingDirectoryMu.RLock()
	getwd := workingDirectory
	workingDirectoryMu.RUnlock()

	cwd, err := getwd()
	if err != nil {
		return location
	}

	relative, err := filepath.Rel(cwd, location)

	if err != nil || relative == emptyString {
		return location
	}

	return relative
}
