// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

type (
	// CoverageRequest describes a go test coverage collection request.
	CoverageRequest = outbound.CoverageRequest
	// PackageRequest describes a Go package listing request.
	PackageRequest = outbound.PackageRequest
	// HTMLOpenRequest describes an HTML coverage report open request.
	HTMLOpenRequest = outbound.HTMLOpenRequest
)
