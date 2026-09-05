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

// Check runs coverage collection and evaluates the configured policy.
func (chk *Checker) Check(parent context.Context, request *Request) (Outcome, error) {
	outcome, err := finishCheck(parent, chk.run, request)

	return outcome, errors.Join(err)
}

func finishCheck(
	parent context.Context,
	run func(context.Context, *Request) (Outcome, error),
	request *Request,
) (Outcome, error) {
	if run == nil {
		return Outcome{}, errCheckerNotConfigured
	}

	outcome, err := run(parent, request)
	if err != nil {
		return Outcome{}, fmt.Errorf(errCoverageCheck, err)
	}

	return outcome, nil
}

// NewChecker creates a coverage checker from its ports.
func NewChecker(coverage outbound.CoverageRunner, packages outbound.PackageCatalog) *Checker {
	return &Checker{run: func(parent context.Context, request *Request) (Outcome, error) {
		return checkCoverage(parent, &checkerPorts{coverage: coverage, packages: packages}, request)
	}}
}

func checkCoverage(parent context.Context, ports *checkerPorts, request *Request) (Outcome, error) {
	if ports.coverage == nil || ports.packages == nil {
		return Outcome{}, errCheckerNotConfigured
	}

	ctx, cancel := context.WithTimeout(parent, request.Timeout)

	defer cancel()

	outcome, err := checkWithTimeout(ctx, ports, request)
	if err != nil {
		return Outcome{}, fmt.Errorf(errCoverageCheck, err)
	}

	return outcome, nil
}

func checkWithTimeout(ctx context.Context, ports *checkerPorts, request *Request) (Outcome, error) {
	collected, listed, err := loadCheckInputs(ctx, ports, request)
	if err != nil {
		return Outcome{}, fmt.Errorf(
			errCoverageCheckWithTimeout,
			timeoutOrError(ctx, request.Timeout, err),
		)
	}

	report := domain.Evaluate(&request.Policy, listed, collected.Blocks)
	domain.AppendTestFailures(&report, listed, &collected)

	return Outcome{
		Report:     report,
		Profile:    collected.Profile,
		TestOutput: collected.TestOutput,
	}, nil
}

func loadCheckInputs(
	ctx context.Context,
	ports *checkerPorts,
	request *Request,
) (coverage domain.Coverage, packages []domain.Package, err error) {
	coverage, err = collectCoverage(ctx, ports.coverage, request)
	if err != nil {
		return domain.Coverage{}, nil, fmt.Errorf(errCoverageCheckWithTimeout, err)
	}

	packages, err = listPackages(ctx, ports.packages, request)
	if err != nil {
		return domain.Coverage{}, nil, fmt.Errorf(errCoverageCheckWithTimeout, err)
	}

	return coverage, packages, nil
}

func collectCoverage(
	ctx context.Context,
	coverage outbound.CoverageRunner,
	request *Request,
) (domain.Coverage, error) {
	collected, err := coverage.Collect(
		ctx,
		&outbound.CoverageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
		},
	)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("collect coverage: %w", err)
	}

	return collected, nil
}

func listPackages(
	ctx context.Context,
	packages outbound.PackageCatalog,
	request *Request,
) ([]domain.Package, error) {
	listed, err := packages.List(
		ctx,
		&outbound.PackageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}

	return listed, nil
}

func timeoutOrError(ctx context.Context, timeout time.Duration, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("coverage check exceeded timeout %s: %w", timeout, ctx.Err())
	}

	return fmt.Errorf(errCoverageCheck, err)
}
