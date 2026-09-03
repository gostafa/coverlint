// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package outbound

import (
	"io"
)

type (
	// CoverageRequest describes a go test coverage collection request.
	CoverageRequest = struct {
		Patterns []string
		TestArgs []string
	}

	// PackageRequest describes a Go package listing request.
	PackageRequest = struct {
		Patterns []string
		TestArgs []string
	}

	// HTMLOpenRequest describes an HTML coverage report open request.
	HTMLOpenRequest = struct {
		Stdout  io.Writer
		Stderr  io.Writer
		Profile []byte
	}
)
