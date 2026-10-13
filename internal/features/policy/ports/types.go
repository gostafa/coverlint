// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package ports defines policy execution dependencies.
package ports

import (
	"context"

	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
)

type (
	// ReportReader reads previously captured failures. Missing files are empty; cancellation is honored.
	ReportReader interface {
		Read(ctx context.Context, path string) (policyinputs.Coverage, error)
	}

	// ArtifactWriter replaces an entire artifact, creating parent directories.
	// Writes occur in order and are not atomic across files. Only safe operations retry.
	ArtifactWriter interface {
		Write(ctx context.Context, path string, data []byte) error
	}

	// CoverageRequest describes a go test coverage collection request.
	CoverageRequest = struct {
		// Patterns selects packages for the operation.
		Patterns []string
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string
		// Imported contains prior test failures without replacing current-run output.
		Imported policyinputs.Coverage
	}

	// InputCollector supplies policy-owned inputs for validated package patterns/test flags.
	// It merges imported failure metadata while retaining current output/profile.
	// Hard failures match ErrCollection; cancellation terminates execution.
	InputCollector interface {
		Load(ctx context.Context, request *CoverageRequest) (policyinputs.Inputs, error)
	}
)
