// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
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
