// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gostafa/coverlint/internal/features/policy/domain"
	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
	"github.com/gostafa/coverlint/internal/features/policy/ports"
)

// NewChecker constructs a coverage use case from its consumer-owned ports.
func NewChecker(collector ports.InputCollector, dependencies ...Dependencies) *Checker {
	active := newExecution(collector, dependencies)

	return &Checker{run: func(ctx context.Context, request *Request) (Outcome, error) {
		return executeCheck(ctx, active, request)
	}}
}

func artifactsFor(request *Request, outcome *Outcome) []artifact {
	return []artifact{
		{
			path:  request.TestResultPath,
			data:  []byte(outcome.TestOutput),
			label: "write test result file",
		},
		{
			path:  request.CoverageResultPath,
			data:  outcome.Profile,
			label: "write coverage result file",
		},
	}
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
		&ports.CoverageRequest{
			Patterns: request.Patterns,
			TestArgs: request.TestArgs,
			Imported: request.TestReport,
		},
	)
	if err != nil {
		return Outcome{}, fmt.Errorf(
			errCoverageCheckWithTimeout,
			timeoutOrError(ctx, request.Timeout, err),
		)
	}

	return evaluateInputs(&inputs, request), nil
}

func evaluateInputs(inputs *policyinputs.Inputs, request *Request) Outcome {
	report := domain.EvaluateInputs(
		&request.Policy,
		&domain.Evaluation{
			Packages:         inputs.Packages,
			Blocks:           inputs.Coverage.Blocks,
			WorkingDirectory: request.WorkingDirectory,
		},
	)
	domain.AppendTestFailures(&report, inputs.Packages, &inputs.Coverage)

	return Outcome{
		Report:     report,
		Profile:    inputs.Coverage.Profile,
		TestOutput: inputs.Coverage.TestOutput,
	}
}

func executeCheck(ctx context.Context, active *execution, request *Request) (Outcome, error) {
	local, err := importReport(ctx, active.dependencies.Reports, request)
	if err != nil {
		return Outcome{}, fmt.Errorf("import report: %w", err)
	}

	outcome, err := checkCoverage(ctx, active.collector, local)
	if err != nil {
		return Outcome{}, fmt.Errorf("check coverage: %w", err)
	}

	err = writeArtifacts(ctx, active.dependencies.Artifacts, artifactsFor(local, &outcome))
	if err != nil {
		return Outcome{}, fmt.Errorf("write result files: %w", err)
	}

	return outcome, nil
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

func importReport(
	ctx context.Context,
	reader ports.ReportReader,
	request *Request,
) (*Request, error) {
	local := *request

	if local.TestResultPath == emptyString {
		return &local, nil
	}

	imported, err := readExistingReport(ctx, reader, local.TestResultPath)
	if err != nil {
		return nil, fmt.Errorf("read test result file: %w", err)
	}

	local.TestReport = imported

	return &local, nil
}

func newExecution(collector ports.InputCollector, dependencies []Dependencies) *execution {
	active := &execution{
		collector:    collector,
		dependencies: Dependencies{Reports: nil, Artifacts: nil},
	}

	if len(dependencies) > zero {
		active.dependencies = dependencies[zero]
	}

	return active
}

func readExistingReport(
	ctx context.Context,
	reader ports.ReportReader,
	path string,
) (policyinputs.Coverage, error) {
	if reader == nil {
		return policyinputs.Coverage{}, errCheckerNotConfigured
	}

	value, err := reader.Read(ctx, path)
	if err != nil {
		return policyinputs.Coverage{}, fmt.Errorf("read existing report: %w", err)
	}

	return value, nil
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

func writeArtifact(ctx context.Context, writer ports.ArtifactWriter, item *artifact) error {
	if item.path == emptyString {
		return nil
	}

	if writer == nil {
		return errCheckerNotConfigured
	}

	err := writer.Write(ctx, item.path, item.data)
	if err != nil {
		return fmt.Errorf("%s: %w", item.label, err)
	}

	return nil
}

func writeArtifacts(ctx context.Context, writer ports.ArtifactWriter, artifacts []artifact) error {
	for i := range artifacts {
		err := writeArtifact(ctx, writer, &artifacts[i])
		if err != nil {
			return fmt.Errorf("write artifact: %w", err)
		}
	}

	return nil
}

// Check runs coverage collection and evaluates the configured policy.
func (chk *Checker) Check(parent context.Context, request *Request) (Outcome, error) {
	outcome, err := finishCheck(parent, chk.run, request)

	return outcome, errors.Join(err)
}
