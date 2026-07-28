package text

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func Diagnostic(result domain.Result, linterName string) string {
	location := result.File
	if location == "" {
		location = result.ImportPath
	} else if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, location); err == nil && rel != "" {
			location = rel
		}
	}
	location = filepath.ToSlash(location)
	return fmt.Sprintf("%s:1:1: %s (%s)", location, result.Message, linterName)
}
