// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverage

import (
	"context"
	"fmt"

	"github.com/gostafa/coverlint/coverlint"
	reportingdomain "github.com/gostafa/coverlint/internal/features/reporting/domain"
)

// Check delegates to the coverlint facade.
func Check(ctx context.Context, input *Config, packagePatterns ...string) (Run, error) {
	run, err := coverlint.Check(ctx, input, packagePatterns...)
	if err != nil {
		return Run{}, fmt.Errorf("Check: %w", err)
	}

	return run, nil
}

// Diagnostic formats a coverage result as a linter diagnostic line.
func Diagnostic(result *Result) string {
	return reportingdomain.Diagnostic(result, Name)
}

// ValidateMinimum delegates to the coverlint facade.
func ValidateMinimum(value float64) error {
	err := coverlint.ValidateMinimum(value)
	if err != nil {
		return fmt.Errorf("validate coverage minimum: %w", err)
	}

	return nil
}
