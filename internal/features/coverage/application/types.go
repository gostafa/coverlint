// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

type (
	// Request contains the inputs for one coverage check.
	Request = struct {
		Policy   domain.Policy
		Patterns []string
		TestArgs []string
		Timeout  time.Duration
	}

	// Outcome contains the report and raw profile from a coverage check.
	Outcome = struct {
		Profile    []byte
		TestOutput string
		Report     domain.Report
	}

	// CheckRunner runs a coverage policy check.
	CheckRunner interface {
		Check(parent context.Context, request *Request) (Outcome, error)
	}

	// Checker coordinates package discovery, coverage collection, and evaluation.
	Checker struct {
		run func(parent context.Context, request *Request) (Outcome, error)
	}

	checkerPorts = struct {
		coverage outbound.CoverageRunner
		packages outbound.PackageCatalog
	}
)
