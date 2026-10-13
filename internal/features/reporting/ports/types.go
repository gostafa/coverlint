// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"context"
	"io"
)

type (
	// HTMLOpenRequest describes an HTML coverage report open request.
	HTMLOpenRequest = struct {
		// Stdout receives command output.
		Stdout io.Writer
		// Stderr receives command diagnostics.
		Stderr io.Writer
		// Profile contains the atomic Go coverprofile.
		Profile []byte
	}

	// HTMLReporter opens an HTML coverage report.
	HTMLReporter interface {
		Open(ctx context.Context, request *HTMLOpenRequest) error
	}
)
