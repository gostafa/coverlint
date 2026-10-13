// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package public

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/gostafa/coverlint/internal/features/reporting/ports"
)

type reporterFunc func(context.Context, *ports.HTMLOpenRequest) error

func (open reporterFunc) Open(ctx context.Context, request *ports.HTMLOpenRequest) error {
	return open(ctx, request)
}

func TestReporterPublishesHTMLAndDiagnosticOperations(t *testing.T) {
	t.Parallel()
	calls := 0
	reporter := &Reporter{
		html: reporterFunc(func(_ context.Context, request *ports.HTMLOpenRequest) error {
			calls++
			if string(request.Profile) != "profile" || request.Stdout != io.Discard {
				t.Fatal("HTML request changed")
			}
			return nil
		}),
	}
	if err := reporter.Open(
		t.Context(),
		&HTMLOpenRequest{Profile: []byte("profile"), Stdout: io.Discard, Stderr: io.Discard},
	); err != nil ||
		calls != 1 {
		t.Fatalf("open=%v calls=%d", err, calls)
	}
	if err := New().Open(t.Context(), &HTMLOpenRequest{}); !errors.Is(err, ports.ErrReporting) {
		t.Fatalf("empty profile=%v", err)
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	got := Diagnostic(
		&Result{ImportPath: "pkg", File: filepath.Join(cwd, "file.go"), Message: "below threshold"},
		"coverlint",
	)
	if got != "file.go:1:1: below threshold (coverlint)" {
		t.Fatal(got)
	}
}
