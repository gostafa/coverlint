// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package outbound

import (
	"context"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

type (
	// CoverageRunner collects coverage for packages.
	CoverageRunner interface {
		Collect(ctx context.Context, request *CoverageRequest) (domain.Coverage, error)
	}

	// PackageCatalog lists package metadata.
	PackageCatalog interface {
		List(ctx context.Context, request *PackageRequest) ([]domain.Package, error)
	}

	// HTMLReporter opens an HTML coverage report.
	HTMLReporter interface {
		Open(ctx context.Context, request *HTMLOpenRequest) error
	}
)
