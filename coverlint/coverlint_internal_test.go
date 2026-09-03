// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

var errBoom = errors.New("boom")

func TestRunOpenWebDelegatesToReporter(t *testing.T) {
	t.Parallel()

	reporter := &fakeHTMLReporter{profile: nil, err: nil}

	err := (Run{
		Report:  Report{Results: nil, Checked: 0, Failed: 0, Skipped: 0},
		profile: []byte("mode: atomic\n"),
		html:    reporter,
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

	err := (Run{
		Report:  Report{Results: nil, Checked: 0, Failed: 0, Skipped: 0},
		profile: nil,
		html:    &fakeHTMLReporter{profile: nil, err: errBoom},
	}).OpenWeb(
		t.Context(),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
	)

	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report: boom") {
		t.Fatalf("error = %v, want wrapped reporter error", err)
	}
}

type fakeHTMLReporter struct {
	err     error
	profile []byte
}

func (f *fakeHTMLReporter) Open(
	_ context.Context,
	profile []byte,
	_, _ io.Writer,
) error {
	f.profile = append([]byte(nil), profile...)

	return f.err
}
