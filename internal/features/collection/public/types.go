// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package public

import (
	"context"

	"github.com/gostafa/coverlint/internal/features/collection/public/inputs"
)

type (
	// Request selects packages and supplies previously captured failures.
	Request = struct {
		// Patterns selects packages for the operation.
		Patterns []string
		// TestArgs supplies additional validated Go test flags.
		TestArgs []string
		// Imported contains prior test failures without replacing current-run output.
		Imported inputs.Coverage
	}

	// Collector publishes raw inputs without exposing collection implementation types.
	Collector interface {
		Load(ctx context.Context, request *Request) (inputs.Inputs, error)
		ReadReport(ctx context.Context, path string) (inputs.Coverage, error)
	}

	// Service composes the collection adapters and use case.
	Service struct {
		load func(context.Context, *Request) (inputs.Inputs, error)
		read func(context.Context, string) (inputs.Coverage, error)
	}
)
