// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package ports defines contracts for report actions and shared reporting results.
package ports

import (
	"context"
	"io"

	"github.com/gostafa/coverlint/internal/features/policy/domain"
)

type (
	// HTMLOpenRequest describes an HTML coverage report open request.
	HTMLOpenRequest = struct {
		Stdout  io.Writer
		Stderr  io.Writer
		Profile []byte
	}

	// HTMLReporter opens an HTML coverage report.
	HTMLReporter interface {
		Open(ctx context.Context, request *HTMLOpenRequest) error
	}

	// Snapshot contains the immutable violations shared across package callbacks.
	Snapshot = struct {
		Violations map[string]domain.Result
	}

	// ResultProvider supplies one shared reporting snapshot or initialization failure.
	ResultProvider interface {
		Get(ctx context.Context) (Snapshot, error)
	}
)
