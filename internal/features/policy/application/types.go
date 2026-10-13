// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"time"

	"github.com/gostafa/coverlint/internal/features/policy/domain"
	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
	"github.com/gostafa/coverlint/internal/features/policy/ports"
)

type (
	execution = struct {
		collector    ports.InputCollector
		dependencies Dependencies
	}

	artifact = struct {
		path  string
		label string
		data  []byte
	}

	// Dependencies supplies optional imported-report and artifact ports.
	Dependencies = struct {
		// Reports supplies imported-report reads.
		Reports ports.ReportReader
		// Artifacts supplies ordered artifact writes.
		Artifacts ports.ArtifactWriter
	}

	// Request contains the inputs for one coverage check.
	Request = struct {
		// WorkingDirectory provides the caller's path-resolution context.
		WorkingDirectory string
		// TestResultPath selects the imported and rewritten test report.
		TestResultPath string
		// CoverageResultPath selects the saved coverage profile.
		CoverageResultPath string
		Policy             domain.Policy
		// Patterns selects packages for the operation.
		Patterns []string
		// TestArgs supplies additional validated Go test flags.
		TestArgs   []string
		TestReport policyinputs.Coverage
		// Timeout bounds coverage execution.
		Timeout time.Duration
	}

	// Outcome contains the report and raw profile from a coverage check.
	Outcome = struct {
		// Profile contains the atomic Go coverprofile.
		Profile []byte
		// TestOutput contains current-run test output.
		TestOutput string
		// Report summarizes coverage outcomes.
		Report domain.Report
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
