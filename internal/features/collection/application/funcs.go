// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"fmt"

	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
)

// NewCollector creates a collector from test execution and package discovery ports.
func NewCollector(coverage ports.CoverageRunner, packages ports.PackageCatalog) *Collector {
	return &Collector{
		run: func(ctx context.Context, request *ports.CoverageRequest) (domain.Inputs, error) {
			return collectInputs(
				ctx,
				&collectorPorts{coverage: coverage, packages: packages},
				request,
			)
		},
	}
}

func collectInputs(
	ctx context.Context,
	collector *collectorPorts,
	request *ports.CoverageRequest,
) (domain.Inputs, error) {
	if collector.coverage == nil || collector.packages == nil {
		return domain.Inputs{}, errCollectorNotConfigured
	}

	inputs, err := loadInputs(ctx, collector, request)
	if err != nil {
		return domain.Inputs{}, fmt.Errorf(errLoadInputs, err)
	}

	return inputs, nil
}

func loadInputs(
	ctx context.Context,
	collector *collectorPorts,
	request *ports.CoverageRequest,
) (domain.Inputs, error) {
	coverage, err := collector.coverage.Collect(ctx, request)
	if err != nil {
		return domain.Inputs{}, fmt.Errorf("collect coverage: %w", err)
	}

	packages, err := collector.packages.List(ctx, &ports.PackageRequest{
		Patterns: request.Patterns,
		TestArgs: request.TestArgs,
	})
	if err != nil {
		return domain.Inputs{}, fmt.Errorf("list packages: %w", err)
	}

	return domain.Inputs{Coverage: coverage, Packages: packages}, nil
}

// Load collects coverage and discovers its package metadata.
func (collector *Collector) Load(
	ctx context.Context,
	request *ports.CoverageRequest,
) (domain.Inputs, error) {
	if collector.run == nil {
		return domain.Inputs{}, errCollectorNotConfigured
	}

	inputs, err := collector.run(ctx, request)
	if err != nil {
		return domain.Inputs{}, fmt.Errorf(errLoadInputs, err)
	}

	return inputs, nil
}
