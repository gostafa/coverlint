// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gostafa/coverlint/internal/features/coverage/application"
	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
	"github.com/gostafa/coverlint/internal/infrastructure/gotool"
)

// Check runs the configured coverage policy against package patterns.
func Check(ctx context.Context, input *Config, packagePatterns ...string) (Run, error) {
	resolved, err := config.Resolve(input, packagePatterns)
	if err != nil {
		return Run{}, fmt.Errorf("resolve coverage config: %w", err)
	}

	run, err := checkResolved(ctx, &resolved)
	if err != nil {
		return Run{}, fmt.Errorf("Check: %w", err)
	}

	return run, nil
}

// OpenWeb opens the standard Go HTML coverage report.
func (run *Run) OpenWeb(ctx context.Context, stdout, stderr io.Writer) error {
	return errors.Join(wrapHTMLCoverage(ctx, run.html, &htmlOpenArgs{
		profile: run.profile,
		stdout:  stdout,
		stderr:  stderr,
	}), htmlReportGuard(run.Report.Checked))
}

func htmlReportGuard(checked int) error {
	if checked < zero {
		return errHTMLAdapterNotConfigured
	}

	return nil
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

// ValidateMinimum checks whether a coverage minimum is allowed.
func ValidateMinimum(value float64) error {
	err := domain.ValidateMinimum(value)
	if err != nil {
		return fmt.Errorf("validate coverage minimum: %w", err)
	}

	return nil
}

func checkResolved(ctx context.Context, resolved *config.Resolved) (Run, error) {
	toolchain := gotool.New()

	outcome, err := application.NewChecker(toolchain, toolchain).Check(
		ctx,
		&application.Request{
			Policy:   resolved.Policy,
			Patterns: resolved.Patterns,
			Timeout:  resolved.Timeout,
			TestArgs: resolved.TestArgs,
		},
	)
	if err != nil {
		return Run{}, fmt.Errorf("check coverage: %w", err)
	}

	return Run{
		Report:  outcome.Report,
		profile: outcome.Profile,
		html:    htmlOpenerFromToolchain(toolchain),
	}, nil
}

func htmlOpenerFromToolchain(toolchain *gotool.Adapter) htmlOpener {
	return func(ctx context.Context, args *htmlOpenArgs) error {
		err := toolchain.Open(ctx, &outbound.HTMLOpenRequest{
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
