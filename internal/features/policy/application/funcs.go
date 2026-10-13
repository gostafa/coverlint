// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	collectiondomain "github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
	"github.com/gostafa/coverlint/internal/features/policy/domain"
)

// NewChecker creates a coverage checker from its ports.
func NewChecker(collector ports.InputCollector) *Checker {
	return &Checker{run: func(parent context.Context, request *Request) (Outcome, error) {
		return checkCoverage(parent, collector, request)
	}}
}

func checkCoverage(
	parent context.Context,
	collector ports.InputCollector,
	request *Request,
) (Outcome, error) {
	if collector == nil {
		return Outcome{}, errCheckerNotConfigured
	}

	ctx, cancel := context.WithTimeout(parent, request.Timeout)

	defer cancel()

	outcome, err := checkWithTimeout(ctx, collector, request)
	if err != nil {
		return Outcome{}, fmt.Errorf(errCoverageCheck, err)
	}

	return outcome, nil
}

func checkWithTimeout(
	ctx context.Context,
	collector ports.InputCollector,
	request *Request,
) (Outcome, error) {
	inputs, err := collector.Load(
		ctx,
		&ports.CoverageRequest{Patterns: request.Patterns, TestArgs: request.TestArgs},
	)
	if err != nil {
		return Outcome{}, fmt.Errorf(
			errCoverageCheckWithTimeout,
			timeoutOrError(ctx, request.Timeout, err),
		)
	}

	return evaluateInputs(&inputs, request), nil
}

func evaluateInputs(inputs *collectiondomain.Inputs, request *Request) Outcome {
	collectiondomain.MergeTestFailures(&inputs.Coverage, &request.TestReport)

	report := domain.Evaluate(&request.Policy, inputs.Packages, inputs.Coverage.Blocks)
	domain.AppendTestFailures(&report, inputs.Packages, &inputs.Coverage)

	return Outcome{
		Report:     report,
		Profile:    inputs.Coverage.Profile,
		TestOutput: inputs.Coverage.TestOutput,
	}
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

func timeoutOrError(ctx context.Context, timeout time.Duration, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf(
			"coverage check exceeded timeout %s: %w",
			timeout,
			errors.Join(ctx.Err(), context.Cause(ctx)),
		)
	}

	return fmt.Errorf(errCoverageCheck, err)
}

// Check runs coverage collection and evaluates the configured policy.
func (chk *Checker) Check(parent context.Context, request *Request) (Outcome, error) {
	outcome, err := finishCheck(parent, chk.run, request)

	return outcome, errors.Join(err)
}
