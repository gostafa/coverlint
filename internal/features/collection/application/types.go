// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package application loads raw coverage and package metadata through collection ports.
package application

import (
	"context"

	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
)

type (
	// Collector coordinates test execution and package discovery.
	Collector struct {
		run func(context.Context, *ports.CoverageRequest) (domain.Inputs, error)
	}

	collectorPorts = struct {
		coverage ports.CoverageRunner
		packages ports.PackageCatalog
	}

	// loadRunner loads raw coverage and package metadata.
	loadRunner interface {
		Load(ctx context.Context, request *ports.CoverageRequest) (domain.Inputs, error)
	}
)
