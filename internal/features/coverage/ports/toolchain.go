// Package ports defines coverage feature boundaries.
package ports

import (
	"context"
	"io"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

// CoverageRequest describes a go test coverage collection request.
type CoverageRequest struct {
	Patterns []string
	TestArgs []string
}

// PackageRequest describes a Go package listing request.
type PackageRequest struct {
	Patterns []string
	TestArgs []string
}

// CoverageRunner collects coverage for packages.
type CoverageRunner interface {
	Collect(ctx context.Context, request CoverageRequest) (domain.Coverage, error)
}

// PackageCatalog lists package metadata.
type PackageCatalog interface {
	List(ctx context.Context, request PackageRequest) ([]domain.Package, error)
}

// HTMLReporter opens an HTML coverage report.
type HTMLReporter interface {
	Open(ctx context.Context, profile []byte, stdout io.Writer, stderr io.Writer) error
}
