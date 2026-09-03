// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

// Type aliases for backward compatibility during migration.
type (
	CoverageRequest = outbound.CoverageRequest
	PackageRequest  = outbound.PackageRequest
	CoverageRunner  = outbound.CoverageRunner
	PackageCatalog  = outbound.PackageCatalog
	HTMLReporter    = outbound.HTMLReporter
)
