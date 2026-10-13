// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package public

import (
	"context"
	"errors"

	"github.com/gostafa/coverlint/internal/features/reporting/adapters/gotool"
	"github.com/gostafa/coverlint/internal/features/reporting/domain"
	"github.com/gostafa/coverlint/internal/infrastructure/environment"
)

// Diagnostic formats a result relative to the process working directory.
func Diagnostic(result *Result, name string) string {
	cwd := environment.WorkingDirectory()
	value := *result

	return domain.Diagnostic(&value, name, cwd)
}

// New composes HTML reporting.
func New() *Reporter { return &Reporter{html: gotool.New()} }

// Open delegates the profile to the feature-owned HTML port.
func (reporter *Reporter) Open(ctx context.Context, request *HTMLOpenRequest) error {
	return errors.Join(reporter.html.Open(ctx, request))
}
