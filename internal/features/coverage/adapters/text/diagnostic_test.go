// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package text_test

import (
	"path/filepath"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/adapters/text"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const coverageBelowMessage = "coverage below 90%"

func TestDiagnosticUsesImportPathWhenFileMissing(t *testing.T) {
	t.Parallel()

	got := text.Diagnostic(
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

	got := text.Diagnostic(
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

	got := text.Diagnostic(
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

func resultForDiagnostic(importPath, file, message string) *domain.Result {
	return &domain.Result{
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
