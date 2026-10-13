// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package collection

import (
	"context"
	"errors"
	"slices"

	"github.com/gostafa/coverlint/internal/features/collection/public"
	collectioninputs "github.com/gostafa/coverlint/internal/features/collection/public/inputs"
	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
	"github.com/gostafa/coverlint/internal/features/policy/ports"
	"github.com/gostafa/coverlint/internal/infrastructure/failure"
)

// New creates a policy adapter from the provider's published contract.
func New(source public.Collector) *Adapter { return &Adapter{source: source} }

func copyCoverage(value *collectioninputs.Coverage) policyinputs.Coverage {
	return policyinputs.Coverage{
		TestOutput:     value.TestOutput,
		Profile:        slices.Clone(value.Profile),
		FailedPackages: slices.Clone(value.FailedPackages),
		TestsFailed:    value.TestsFailed,
		Blocks:         slices.Clone(value.Blocks),
		Failures:       slices.Clone(value.Failures),
	}
}

func copyInputs(value *collectioninputs.Inputs) policyinputs.Inputs {
	result := policyinputs.Inputs{Coverage: copyCoverage(&value.Coverage), Packages: nil}

	for i := range value.Packages {
		result.Packages = append(result.Packages, copyPackage(&value.Packages[i]))
	}

	return result
}

func copyPackage(value *collectioninputs.Package) policyinputs.Package {
	return policyinputs.Package{
		ImportPath: value.ImportPath,
		Dir:        value.Dir,
		FirstFile:  value.FirstFile,
		Files:      slices.Clone(value.Files),
	}
}

func mappedCoverage(value *collectioninputs.Coverage, cause error) (policyinputs.Coverage, error) {
	if cause != nil {
		return policyinputs.Coverage{}, errors.Join(failure.Translate(cause, ports.ErrCollection))
	}

	return copyCoverage(value), nil
}

func mappedInputs(value *collectioninputs.Inputs, cause error) (policyinputs.Inputs, error) {
	if cause != nil {
		return policyinputs.Inputs{}, errors.Join(failure.Translate(cause, ports.ErrCollection))
	}

	return copyInputs(value), nil
}

// Load maps collected inputs into independently owned policy data.
func (adapter *Adapter) Load(
	ctx context.Context,
	request *ports.CoverageRequest,
) (policyinputs.Inputs, error) {
	value, err := adapter.source.Load(
		ctx,
		&public.Request{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
			Imported: copyCoverage(&request.Imported),
		},
	)
	result, err := mappedInputs(&value, err)

	return result, errors.Join(err)
}

// Read imports saved failures through the published collection contract.
func (adapter *Adapter) Read(ctx context.Context, path string) (policyinputs.Coverage, error) {
	value, err := adapter.source.ReadReport(ctx, path)
	result, err := mappedCoverage(&value, err)

	return result, errors.Join(err)
}
