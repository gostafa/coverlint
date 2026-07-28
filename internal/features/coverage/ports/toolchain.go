package ports

import (
	"context"
	"io"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

type CoverageRequest struct {
	Patterns []string
	TestArgs []string
}

type PackageRequest struct {
	Patterns []string
	TestArgs []string
}

type CoverageRunner interface {
	Collect(context.Context, CoverageRequest) (domain.Coverage, error)
}

type PackageCatalog interface {
	List(context.Context, PackageRequest) ([]domain.Package, error)
}

type HTMLReporter interface {
	Open(context.Context, []byte, io.Writer, io.Writer) error
}
