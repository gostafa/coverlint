// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain_test

import (
	"path/filepath"
	"testing"

	coveragedomain "github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/reporting/domain"
)

const coverageBelowMessage = "coverage below 90%"

func TestDiagnosticUsesImportPathWhenFileMissing(t *testing.T) {
	t.Parallel()

	got := domain.Diagnostic(
		resultForDiagnostic("example.com/pkg", "", coverageBelowMessage),
		"coverlint",
	)

	want := "example.com/pkg:1:1: coverage below 90% (coverlint)"

	if got != want {
		t.Fatalf("Diagnostic() = %q, want %q", got, want)
	}
}

func TestDiagnosticFallsBackWhenFileCannotBeRelativized(t *testing.T) {
	t.Parallel()

	got := domain.Diagnostic(
		resultForDiagnostic("", string([]byte{0}), coverageBelowMessage),
		"coverlint",
	)

	want := "\x00:1:1: coverage below 90% (coverlint)"

	if got != want {
		t.Fatalf("Diagnostic() = %q, want %q", got, want)
	}
}

func TestDiagnosticRelativizesFile(t *testing.T) {
	t.Parallel()

	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	got := domain.Diagnostic(
		resultForDiagnostic(
			"",
			filepath.Join(cwd, "internal", "pkg", "file.go"),
			coverageBelowMessage,
		),
		"coverlint",
	)

	want := "internal/pkg/file.go:1:1: coverage below 90% (coverlint)"

	if got != want {
		t.Fatalf("Diagnostic() = %q, want %q", got, want)
	}
}

func resultForDiagnostic(importPath, file, message string) coveragedomain.Result {
	return coveragedomain.Result{
		ImportPath: importPath,
		File:       file,
		Rule:       nil,
		Coverage:   0,
		Statements: 0,
		Covered:    0,
		Skipped:    false,
		Violation:  false,
		Message:    message,
	}
}
