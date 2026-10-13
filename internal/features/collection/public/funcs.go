// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package public

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/gostafa/coverlint/internal/features/collection/adapters/gotool"
	"github.com/gostafa/coverlint/internal/features/collection/application"
	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
	"github.com/gostafa/coverlint/internal/features/collection/public/inputs"
	"github.com/gostafa/coverlint/internal/infrastructure/failure"
)

// New composes Go test execution, package discovery, and report import.
func New() *Service {
	tool := gotool.New()
	collector := application.NewCollector(tool, tool)

	return &Service{load: collectionLoader(collector), read: readPublishedReport}
}

func collectionLoader(
	collector ports.InputCollector,
) func(context.Context, *Request) (inputs.Inputs, error) {
	return func(ctx context.Context, request *Request) (inputs.Inputs, error) {
		value, err := collector.Load(
			ctx,
			&ports.CoverageRequest{
				Patterns: slices.Clone(request.Patterns),
				TestArgs: slices.Clone(request.TestArgs),
				Imported: copyCoverage(&request.Imported),
			},
		)
		if err != nil {
			return inputs.Inputs{}, errors.Join(
				failure.Translate(fmt.Errorf("collect inputs: %w", err), ports.ErrCollection),
			)
		}

		return copyInputs(&value), nil
	}
}

func copyCoverage(value *domain.Coverage) inputs.Coverage {
	return inputs.Coverage{
		TestOutput:     value.TestOutput,
		Profile:        slices.Clone(value.Profile),
		FailedPackages: slices.Clone(value.FailedPackages),
		TestsFailed:    value.TestsFailed,
		Blocks:         slices.Clone(value.Blocks),
		Failures:       slices.Clone(value.Failures),
	}
}

func copyInputs(value *domain.Inputs) inputs.Inputs {
	result := inputs.Inputs{Coverage: copyCoverage(&value.Coverage), Packages: nil}

	for i := range value.Packages {
		result.Packages = append(result.Packages, copyPackage(&value.Packages[i]))
	}

	return result
}

func copyPackage(value *domain.Package) inputs.Package {
	return inputs.Package{
		ImportPath: value.ImportPath,
		Dir:        value.Dir,
		FirstFile:  value.FirstFile,
		Files:      slices.Clone(value.Files),
	}
}

func readPublishedReport(ctx context.Context, path string) (inputs.Coverage, error) {
	value, err := gotool.ReadTestReportContext(ctx, path)
	if err != nil {
		return inputs.Coverage{}, errors.Join(
			failure.Translate(fmt.Errorf("read report: %w", err), ports.ErrCollection),
		)
	}

	return copyCoverage(&value), nil
}

// Load supplies fresh inputs and merged failure metadata.
func (service *Service) Load(ctx context.Context, request *Request) (inputs.Inputs, error) {
	value, err := service.load(ctx, request)

	return value, errors.Join(err)
}

// ReadReport imports saved failures without rewriting the source report.
func (service *Service) ReadReport(ctx context.Context, path string) (inputs.Coverage, error) {
	value, err := service.read(ctx, path)

	return value, errors.Join(err)
}
