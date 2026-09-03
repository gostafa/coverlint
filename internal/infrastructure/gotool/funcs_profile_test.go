// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/infrastructure/gotool"
)

var errReadFailed = errors.New("read failed")

func TestParseProfileKeepsBlockPosition(t *testing.T) {
	t.Parallel()

	blocks, err := gotool.ParseProfile(
		strings.NewReader("mode: atomic\nexample.com/app/a.go:10.1,12.2 3 1\n"),
	)
	if err != nil {
		t.Fatalf("parseProfile: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1", len(blocks))
	}

	if blocks[0].Position != "10.1,12.2" {
		t.Fatalf("Position = %q, want profile range", blocks[0].Position)
	}
}

func TestParseProfileSkipsBlankLines(t *testing.T) {
	t.Parallel()

	blocks, err := gotool.ParseProfile(
		strings.NewReader("mode: atomic\n\nexample.com/app/a.go:10.1,12.2 3 0\n"),
	)
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}

	if len(blocks) != 1 || blocks[0].Covered {
		t.Fatalf("blocks = %#v, want one uncovered block", blocks)
	}
}

func TestParseProfileRejectsBadInput(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "coverage profile is empty"},
		{name: "missing mode", input: "not mode\n", want: "coverage profile has no mode header"},
		{name: "missing colon", input: "mode: atomic\nbad\n", want: "coverage profile line is malformed: 2"},
		{name: "bad position", input: "mode: atomic\nfile.go:bad 1 1\n", want: "coverage profile line is malformed: 2"},
		{
			name:  "bad statements",
			input: "mode: atomic\nfile.go:1.1,2.1 nope 1\n",
			want:  "coverage profile line has invalid statement count: 2",
		},
		{
			name:  "negative statements",
			input: "mode: atomic\nfile.go:1.1,2.1 -1 1\n",
			want:  "coverage profile line has invalid statement count: 2",
		},
		{
			name:  "bad count",
			input: "mode: atomic\nfile.go:1.1,2.1 1 nope\n",
			want:  "coverage profile line has invalid execution count: 2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := gotool.ParseProfile(strings.NewReader(test.input))

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestParseProfileWrapsScannerError(t *testing.T) {
	t.Parallel()

	_, err := gotool.ParseProfile(errReader{})

	if err == nil || !strings.Contains(err.Error(), "read coverage profile") {
		t.Fatalf("error = %v, want read wrapper", err)
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errReadFailed
}
