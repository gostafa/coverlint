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
		// Patterns selects packages for the operation.
		Patterns []string
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string
		// Imported contains prior test failures without replacing current-run output.
		Imported domain.Coverage
	}

	// PackageRequest describes a Go package listing request.
	PackageRequest = struct {
		// Patterns selects packages for the operation.
		Patterns []string
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string
	}

	// InputCollector loads fresh coverage and package metadata, merging imported failures.
	// Current output/profile remain authoritative. Calls honor cancellation and are independent.
	InputCollector interface {
		Load(ctx context.Context, request *CoverageRequest) (domain.Inputs, error)
	}

	// CoverageRunner collects fresh coverage for validated package patterns/test flags.
	// Failed tests with usable profiles are data; hard failures match ErrCollection.
	// Cancellation stops execution; a process that has started is never rerun.
	CoverageRunner interface {
		Collect(
			ctx context.Context,
			request *CoverageRequest,
		) (domain.Coverage, error)
	}

	// PackageCatalog lists package metadata for the same validated build context.
	// Failures match ErrPackageList. Successful process startup is not retried.
	PackageCatalog interface {
		List(
			ctx context.Context,
			request *PackageRequest,
		) ([]domain.Package, error)
	}
)
