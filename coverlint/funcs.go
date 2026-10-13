// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	collectionpublic "github.com/gostafa/coverlint/internal/features/collection/public"
	"github.com/gostafa/coverlint/internal/features/policy/adapters/collection"
	"github.com/gostafa/coverlint/internal/features/policy/adapters/config"
	"github.com/gostafa/coverlint/internal/features/policy/adapters/files"
	"github.com/gostafa/coverlint/internal/features/policy/application"
	"github.com/gostafa/coverlint/internal/features/policy/domain"
	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
	"github.com/gostafa/coverlint/internal/features/policy/ports"
	reportingpublic "github.com/gostafa/coverlint/internal/features/reporting/public"
)

// Check runs the configured coverage policy against package patterns.
func Check(ctx context.Context, input *Config, packagePatterns ...string) (Run, error) {
	resolved, err := config.Resolve(configInput(input), packagePatterns)
	if err != nil {
		return Run{}, errors.Join(publicError("resolve coverage config", err, ErrInvalidConfig))
	}

	run, err := checkResolved(ctx, &resolved)
	if err != nil {
		return Run{}, errors.Join(publicCheckError(err))
	}

	return run, nil
}

// ValidateMinimum checks whether a coverage minimum is allowed.
func ValidateMinimum(value float64) error {
	err := domain.ValidateMinimum(value)
	if err != nil {
		return errors.Join(publicError("validate coverage minimum", err, ErrInvalidConfig))
	}

	return nil
}

func checkResolved(ctx context.Context, resolved *config.Resolved) (Run, error) {
	source := collection.New(collectionpublic.New())
	checker := application.NewChecker(
		source,
		application.Dependencies{Reports: source, Artifacts: new(files.Adapter)},
	)

	outcome, err := checker.Check(ctx, checkerRequest(resolved))
	if err != nil {
		return Run{}, fmt.Errorf("check coverage: %w", err)
	}

	return runFromOutcome(&outcome, reportingpublic.New()), nil
}

func checkerRequest(resolved *config.Resolved) *application.Request {
	return &application.Request{
		Policy:             resolved.Policy,
		Patterns:           resolved.Patterns,
		Timeout:            resolved.Timeout,
		TestArgs:           resolved.TestArgs,
		TestReport:         emptyCoverage(),
		WorkingDirectory:   resolved.WorkingDirectory,
		TestResultPath:     resolved.TestResultPath,
		CoverageResultPath: resolved.CoverageResultPath,
	}
}

func configInput(input *Config) *config.Config {
	if input == nil {
		return nil
	}

	result := &config.Config{
		Timeout:            input.Timeout,
		TestResultPath:     input.TestResultPath,
		CoverageResultPath: input.CoverageResultPath,
		Packages:           slices.Clone(input.Packages),
		TestArgs:           slices.Clone(input.TestArgs),
		Rules:              nil,
	}

	for i := range input.Rules {
		result.Rules = append(result.Rules, domain.Rule(input.Rules[i]))
	}

	return result
}

func emptyCoverage() policyinputs.Coverage {
	var value policyinputs.Coverage

	return value
}

func htmlOpenerFromToolchain(toolchain htmlReporter) htmlOpener {
	return func(ctx context.Context, args *htmlOpenArgs) error {
		err := toolchain.Open(ctx, &reportingpublic.HTMLOpenRequest{
			Profile: args.profile,
			Stdout:  args.stdout,
			Stderr:  args.stderr,
		})
		if err != nil {
			return fmt.Errorf(errOpenHTMLReport, err)
		}

		return nil
	}
}

func htmlReportGuard(checked int) error {
	if checked < zero {
		return errHTMLAdapterNotConfigured
	}

	return nil
}

func matchesPublicError(failure *boundaryDetails, target error) bool {
	return errors.Is(target, failure.category) ||
		(failure.canceled && errors.Is(target, context.Canceled)) ||
		(failure.deadline && errors.Is(target, context.DeadlineExceeded))
}

func openRunHTML(ctx context.Context, open htmlOpener, args *htmlOpenArgs) error {
	err := errors.Join(wrapHTMLCoverage(ctx, open, args), htmlReportGuard(args.checked))
	if err != nil {
		return errors.Join(publicError("open HTML coverage report", err, ErrReporting))
	}

	return nil
}

func publicCheckError(err error) error {
	category := ErrCollection

	if errors.Is(err, ports.ErrResultWrite) {
		category = ErrResultWrite
	}

	return errors.Join(publicError("Check", err, category))
}

func publicError(operation string, cause, category error) error {
	details := boundaryDetails{
		message:  operation + ": " + cause.Error(),
		category: category,
		canceled: errors.Is(cause, context.Canceled),
		deadline: errors.Is(cause, context.DeadlineExceeded),
	}

	return boundaryError(func() boundaryDetails { return details })
}

func publicReport(value *domain.Report) Report {
	result := Report{
		Checked: value.Checked,
		Failed:  value.Failed,
		Skipped: value.Skipped,
		Results: nil,
	}

	if value.Results != nil {
		result.Results = make([]Result, zero, len(value.Results))
	}

	for i := range value.Results {
		result.Results = append(result.Results, publicResult(&value.Results[i]))
	}

	return result
}

func publicResult(value *domain.Result) Result {
	result := Result{
		ImportPath: value.ImportPath,
		File:       value.File,
		Message:    value.Message,
		Coverage:   value.Coverage,
		Statements: value.Statements,
		Covered:    value.Covered,
		Rule:       nil,
		Skipped:    value.Skipped,
		Violation:  value.Violation,
	}

	if value.Rule != nil {
		rule := Rule(*value.Rule)

		result.Rule = &rule
	}

	return result
}

func runFromOutcome(outcome *application.Outcome, toolchain htmlReporter) Run {
	return Run{
		Report:  publicReport(&outcome.Report),
		profile: outcome.Profile,
		html:    htmlOpenerFromToolchain(toolchain),
	}
}

func wrapHTMLCoverage(ctx context.Context, open htmlOpener, args *htmlOpenArgs) error {
	if open == nil {
		return errHTMLAdapterNotConfigured
	}

	err := open(ctx, args)
	if err != nil {
		return fmt.Errorf(errOpenHTMLReport, err)
	}

	return nil
}

// Error returns the public diagnostic.
func (failure boundaryError) Error() string { return failure().message }

// Is matches public categories and standard cancellation identities.
func (failure boundaryError) Is(target error) bool {
	details := failure()

	return matchesPublicError(&details, target)
}

// Matches reports whether the rule pattern matches importPath.
func (rule Rule) Matches(importPath string) bool {
	return (domain.Rule{Pattern: rule.Pattern, Min: rule.Min}).Matches(importPath)
}

// MinCoverage returns the minimum acceptable coverage fraction.
func (rule Rule) MinCoverage() float64 { return rule.Min }

// OpenWeb opens the standard Go HTML coverage report.
func (run *Run) OpenWeb(ctx context.Context, stdout, stderr io.Writer) error {
	return errors.Join(
		openRunHTML(
			ctx,
			run.html,
			&htmlOpenArgs{
				profile: run.profile,
				stdout:  stdout,
				stderr:  stderr,
				checked: run.Report.Checked,
			},
		),
	)
}

// PatternValue returns the glob pattern.
func (rule Rule) PatternValue() string { return rule.Pattern }

// String returns the pattern and minimum as "pattern:min".
func (rule Rule) String() string { return (domain.Rule{Pattern: rule.Pattern, Min: rule.Min}).String() }
