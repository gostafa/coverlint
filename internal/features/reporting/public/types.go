// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package public publishes reporting operations without implementation types.
package public

import (
	"context"
	"io"

	"github.com/gostafa/coverlint/internal/features/reporting/ports"
)

type (
	// HTMLReporter is the published capability for opening coverage profiles.
	HTMLReporter interface {
		Open(ctx context.Context, request *HTMLOpenRequest) error
	}

	// Result is the diagnostic input contract.
	Result = struct {
		// ImportPath identifies the Go package.
		ImportPath string
		// File locates the diagnostic or coverage block source.
		File string
		// Message contains the user-facing diagnostic.
		Message string
	}

	// HTMLOpenRequest contains a profile and the caller's output streams.
	HTMLOpenRequest = struct {
		// Stdout receives command output.
		Stdout io.Writer
		// Stderr receives command diagnostics.
		Stderr io.Writer
		// Profile contains the atomic Go coverprofile.
		Profile []byte
	}

	// Reporter delegates to the feature-owned HTML port.
	Reporter struct{ html ports.HTMLReporter }
)
