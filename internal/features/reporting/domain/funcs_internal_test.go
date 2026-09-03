// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"errors"
	"testing"
)

var errWorkingDirectory = errors.New("working directory unavailable")

func TestRelativeLocationFallsBackWhenWorkingDirectoryFails(t *testing.T) {
	previous := workingDirectory

	workingDirectoryMu.Lock()
	workingDirectory = func() (string, error) {
		return "", errWorkingDirectory
	}
	workingDirectoryMu.Unlock()

	t.Cleanup(func() {
		workingDirectoryMu.Lock()
		workingDirectory = previous
		workingDirectoryMu.Unlock()
	})

	const location = "/repo/pkg/file.go"

	if got := relativeLocation(location); got != location {
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
