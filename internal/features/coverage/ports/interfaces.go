// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

type (
	// CoverageRunner collects coverage for packages.
	CoverageRunner = outbound.CoverageRunner
	// PackageCatalog lists package metadata.
	PackageCatalog = outbound.PackageCatalog
	// HTMLReporter opens an HTML coverage report.
	HTMLReporter = outbound.HTMLReporter
)
