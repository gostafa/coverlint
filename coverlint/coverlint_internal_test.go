// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
	"github.com/gostafa/coverlint/internal/infrastructure/gotool"
)

var errBoom = errors.New("boom")

func TestRunOpenWebDelegatesToReporter(t *testing.T) {
	t.Parallel()

	reporter := &fakeHTMLReporter{profile: nil, err: nil}

	err := (&Run{
		Report:  Report{Results: nil, Checked: 0, Failed: 0, Skipped: 0},
		profile: []byte("mode: atomic\n"),
		html:    htmlOpenerFromReporter(reporter),
	}).OpenWeb(
		t.Context(),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
	)
	if err != nil {
		t.Fatalf("OpenWeb: %v", err)
	}

	if string(reporter.profile) != "mode: atomic\n" {
		t.Fatalf("profile = %q", reporter.profile)
	}
}

func TestRunOpenWebWrapsReporterError(t *testing.T) {
	t.Parallel()

	err := (&Run{
		Report:  Report{Results: nil, Checked: 0, Failed: 0, Skipped: 0},
		profile: nil,
		html:    htmlOpenerFromReporter(&fakeHTMLReporter{profile: nil, err: errBoom}),
	}).OpenWeb(
		t.Context(),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
	)

	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report: boom") {
		t.Fatalf("error = %v, want wrapped reporter error", err)
	}
}

func TestHTMLReportGuardRejectsNegativeChecked(t *testing.T) {
	t.Parallel()

	err := htmlReportGuard(-1)

	if !errors.Is(err, errHTMLAdapterNotConfigured) {
		t.Fatalf("error = %v, want %v", err, errHTMLAdapterNotConfigured)
	}
}

func TestHTMLOpenerFromToolchainSuccessAndOpenError(t *testing.T) {
	t.Setenv("BROWSER", "true")

	open := htmlOpenerFromToolchain(gotool.New())
	profile := coverageProfileForHTML(t)

	err := open(t.Context(), &htmlOpenArgs{
		profile: profile,
		stdout:  io.Discard,
		stderr:  io.Discard,
	})
	if err != nil {
		t.Fatalf("htmlOpenerFromToolchain success: %v", err)
	}

	err = open(t.Context(), &htmlOpenArgs{
		profile: nil,
		stdout:  io.Discard,
		stderr:  io.Discard,
	})

	if err == nil || !strings.Contains(err.Error(), "coverage profile is empty") {
		t.Fatalf("error = %v, want empty profile", err)
	}
}

func TestCheckWrapsResolvedCheckError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Check(
		ctx,
		&Config{
			Rules:    []Rule{{Pattern: "**", Min: 0.90}},
			Exclude:  nil,
			Packages: nil,
			Timeout:  time.Minute.String(),
			TestArgs: nil,
		},
		".",
	)

	if err == nil || !strings.Contains(err.Error(), "Check:") {
		t.Fatalf("error = %v, want Check wrapper", err)
	}
}

func coverageProfileForHTML(t *testing.T) []byte {
	t.Helper()

	dir := writeInternalCoverageFixture(t)

	coverage, err := gotool.New().Collect(
		t.Context(),
		&outbound.CoverageRequest{Patterns: []string{dir}, TestArgs: []string{"-run", "TestAdd"}},
	)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	return coverage.Profile
}

func writeInternalCoverageFixture(t *testing.T) string {
	t.Helper()

	temp := t.TempDir()
	suffix := filepath.Base(filepath.Dir(temp)) + "-" + filepath.Base(temp)
	dir := filepath.Join(".", "coverlint-fixture-"+strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+"-"+suffix)

	err := os.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Cleanup(func() {
		removeErr := os.RemoveAll(dir)
		if removeErr != nil {
			t.Fatalf("RemoveAll %s: %v", dir, removeErr)
		}
	})

	writeInternalFile(t, dir, "calc.go", "package fixture\n\nfunc Add(a, b int) int { return a + b }\n")
	writeInternalFile(t, dir, "calc_test.go", `package fixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`)

	return "./" + filepath.Base(dir)
}

func writeInternalFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	if err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

func htmlOpenerFromReporter(reporter *fakeHTMLReporter) htmlOpener {
	return func(ctx context.Context, args *htmlOpenArgs) error {
		return reporter.Open(ctx, &outbound.HTMLOpenRequest{
			Profile: args.profile,
			Stdout:  args.stdout,
			Stderr:  args.stderr,
		})
	}
}

type fakeHTMLReporter struct {
	err     error
	profile []byte
}

func (f *fakeHTMLReporter) Open(
	_ context.Context,
	request *outbound.HTMLOpenRequest,
) error {
	f.profile = append([]byte(nil), request.Profile...)

	return f.err
}
