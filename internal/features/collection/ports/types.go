// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package ports defines contracts for coverage collection and report actions.
package ports

import (
	"context"

	"github.com/gostafa/coverlint/internal/features/collection/domain"
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

	// InputCollector loads coverage and package metadata for policy evaluation.
	InputCollector interface {
		Load(ctx context.Context, request *CoverageRequest) (domain.Inputs, error)
	}

	// CoverageRunner collects coverage for packages.
	CoverageRunner interface {
		Collect(
			ctx context.Context,
			request *CoverageRequest,
		) (domain.Coverage, error)
	}

	// PackageCatalog lists package metadata.
	PackageCatalog interface {
		List(
			ctx context.Context,
			request *PackageRequest,
		) ([]domain.Package, error)
	}
)
