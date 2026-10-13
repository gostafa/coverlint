// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package application applies coverage policy to collected inputs.
package application

import (
	"context"
	"time"

	collectiondomain "github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/policy/domain"
)

type (
	// Request contains the inputs for one coverage check.
	Request = struct {
		Policy     domain.Policy
		Patterns   []string
		TestArgs   []string
		TestReport collectiondomain.Coverage
		Timeout    time.Duration
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

	// Checker evaluates policy using a coverage input collector.
	Checker struct {
		run func(parent context.Context, request *Request) (Outcome, error)
	}
)
