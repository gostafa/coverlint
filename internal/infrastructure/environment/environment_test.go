// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package environment

import (
	"errors"
	"testing"
)

func TestWorkingDirectoryAndUnavailableFallback(t *testing.T) {
	t.Parallel()
	if WorkingDirectory() == "" {
		t.Fatal("test working directory unavailable")
	}
	if value := workingDirectoryWith(
		func() (string, error) { return "", errors.New("directory unavailable") },
	); value != "" {
		t.Fatal(value)
	}
	if value := workingDirectoryWith(
		func() (string, error) { return "/repo", nil },
	); value != "/repo" {
		t.Fatal(value)
	}
}
