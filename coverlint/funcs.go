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
	"github.com/gostafa/coverlint/internal/infrastructure/gotool"
)

var errHTMLAdapterNotConfigured = errors.New("HTML coverage adapter is not configured")

// Check runs the configured coverage policy against package patterns.
func Check(ctx context.Context, input Config, packagePatterns ...string) (Run, error) {
	resolved, err := config.Resolve(input, packagePatterns)
	if err != nil {
		return Run{}, fmt.Errorf("resolve coverage config: %w", err)
	}

	toolchain := gotool.New()

	outcome, err := application.NewChecker(toolchain, toolchain).Check(ctx, application.Request{
		Policy:   resolved.Policy,
		Patterns: resolved.Patterns,
		Timeout:  resolved.Timeout,
		TestArgs: resolved.TestArgs,
	})
	if err != nil {
		return Run{}, fmt.Errorf("check coverage: %w", err)
	}

	return Run{Report: outcome.Report, profile: outcome.Profile, html: toolchain}, nil
}

// ValidateMinimum checks whether a coverage minimum is allowed.
func ValidateMinimum(value float64) error {
	err := domain.ValidateMinimum(value)
	if err != nil {
		return fmt.Errorf("validate coverage minimum: %w", err)
	}

	return nil
}

// OpenWeb opens the standard Go HTML coverage report.
func (r Run) OpenWeb(ctx context.Context, stdout, stderr io.Writer) error {
	if r.html == nil {
		return errHTMLAdapterNotConfigured
	}

	err := r.html.Open(ctx, r.profile, stdout, stderr)
	if err != nil {
		return fmt.Errorf("open HTML coverage report: %w", err)
	}

	return nil
}
