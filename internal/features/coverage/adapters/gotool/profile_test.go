package gotool_test

import (
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/adapters/gotool"
)

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
