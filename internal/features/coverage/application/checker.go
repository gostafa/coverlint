// Package application orchestrates coverage collection and policy evaluation.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
)

var errCheckerNotConfigured = errors.New("coverage checker is not configured")

// Request contains the inputs for one coverage check.
type Request struct {
	Policy   domain.Policy
	Patterns []string
	Timeout  time.Duration
	TestArgs []string
}

// Outcome contains the report and raw profile from a coverage check.
type Outcome struct {
	Report  domain.Report
	Profile []byte
}

// Checker coordinates package discovery, coverage collection, and evaluation.
type Checker struct {
	coverage ports.CoverageRunner
	packages ports.PackageCatalog
}

// NewChecker creates a coverage checker from its ports.
func NewChecker(coverage ports.CoverageRunner, packages ports.PackageCatalog) *Checker {
	return &Checker{coverage: coverage, packages: packages}
}

// Check runs coverage collection and evaluates the configured policy.
func (c *Checker) Check(parent context.Context, request Request) (Outcome, error) {
	if c == nil || c.coverage == nil || c.packages == nil {
		return Outcome{}, errCheckerNotConfigured
	}

	ctx, cancel := context.WithTimeout(parent, request.Timeout)
	defer cancel()

	coverage, err := c.coverage.Collect(
		ctx,
		ports.CoverageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return Outcome{}, fmt.Errorf(
				"coverage check exceeded timeout %s: %w",
				request.Timeout,
				ctx.Err(),
			)
		}

		return Outcome{}, fmt.Errorf("collect coverage: %w", err)
	}

	packages, err := c.packages.List(
		ctx,
		ports.PackageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return Outcome{}, fmt.Errorf(
				"coverage check exceeded timeout %s: %w",
				request.Timeout,
				ctx.Err(),
			)
		}

		return Outcome{}, fmt.Errorf("list packages: %w", err)
	}

	return Outcome{
		Report:  request.Policy.Evaluate(packages, coverage.Blocks),
		Profile: coverage.Profile,
	}, nil
}
