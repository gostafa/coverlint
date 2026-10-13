// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"errors"
	"testing"
)

var errWorkingDirectory = errors.New("working directory unavailable")

func TestRelativeLocationFallsBackWhenWorkingDirectoryFails(t *testing.T) {
	t.Parallel()

	const location = "/repo/pkg/file.go"

	got := relativeLocationWith(location, func() (string, error) {
		return "", errWorkingDirectory
	})

	if got != location {
		t.Fatalf("relativeLocation() = %q, want %q", got, location)
	}
}

func TestRelativeLocationFallsBackWhenRelFails(t *testing.T) {
	t.Parallel()

	const location = ":"

	if got := relativeLocation(location); got != location {
		t.Fatalf("relativeLocation() = %q, want %q", got, location)
	}
}
