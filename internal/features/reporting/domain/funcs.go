// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"os"
	"path/filepath"

	coveragedomain "github.com/gostafa/coverlint/internal/features/coverage/domain"
)

// Diagnostic formats a coverage result as a linter diagnostic line.
func Diagnostic(result coveragedomain.Result, linterName string) string {
	location := result.File

	if location == "" {
		location = result.ImportPath
	} else {
		location = relativeLocation(location)
	}

	location = filepath.ToSlash(location)

	return fmt.Sprintf("%s:1:1: %s (%s)", location, result.Message, linterName)
}

func relativeLocation(location string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return location
	}

	relative, err := filepath.Rel(cwd, location)

	if err != nil || relative == "" {
		return location
	}

	return relative
}
