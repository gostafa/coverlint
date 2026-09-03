// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
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
	coverage outbound.CoverageRunner
	packages outbound.PackageCatalog
}

// NewChecker creates a coverage checker from its ports.
func NewChecker(coverage outbound.CoverageRunner, packages outbound.PackageCatalog) *Checker {
	return &Checker{coverage: coverage, packages: packages}
}

// Check runs coverage collection and evaluates the configured policy.
func (c *Checker) Check(parent context.Context, request Request) (Outcome, error) {
	if !c.configured() {
		return Outcome{}, errCheckerNotConfigured
	}

	ctx, cancel := context.WithTimeout(parent, request.Timeout)

	defer cancel()

	coverage, err := c.collect(ctx, request)
	if err != nil {
		return Outcome{}, timeoutOrError(ctx, request.Timeout, err)
	}

	packages, err := c.list(ctx, request)
	if err != nil {
		return Outcome{}, timeoutOrError(ctx, request.Timeout, err)
	}

	return Outcome{
		Report:  request.Policy.Evaluate(packages, coverage.Blocks),
		Profile: coverage.Profile,
	}, nil
}

func (c *Checker) configured() bool {
	return c != nil && c.coverage != nil && c.packages != nil
}

func (c *Checker) collect(ctx context.Context, request Request) (domain.Coverage, error) {
	coverage, err := c.coverage.Collect(
		ctx,
		outbound.CoverageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
		},
	)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("collect coverage: %w", err)
	}

	return coverage, nil
}

func (c *Checker) list(ctx context.Context, request Request) ([]domain.Package, error) {
	packages, err := c.packages.List(
		ctx,
		outbound.PackageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}

	return packages, nil
}

func timeoutOrError(ctx context.Context, timeout time.Duration, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("coverage check exceeded timeout %s: %w", timeout, ctx.Err())
	}

	return err
}
