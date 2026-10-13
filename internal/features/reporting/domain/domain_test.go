// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"testing"
)

func TestRelativeLocationFallsBackWhenWorkingDirectoryFails(t *testing.T) {
	t.Parallel()

	const location = "/repo/pkg/file.go"

	got := relativeLocation(location, "")

	if got != location {
		t.Fatalf("relativeLocation() = %q, want %q", got, location)
	}
}

func TestRelativeLocationFallsBackWhenRelFails(t *testing.T) {
	t.Parallel()

	const location = ":"

	if got := relativeLocation(location, "/repo"); got != location {
		t.Fatalf("relativeLocation() = %q, want %q", got, location)
	}
}
