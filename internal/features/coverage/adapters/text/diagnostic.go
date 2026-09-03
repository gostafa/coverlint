// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package text

import (
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	reportingdomain "github.com/gostafa/coverlint/internal/features/reporting/domain"
)

// Diagnostic formats a coverage result as a linter diagnostic line.
func Diagnostic(result *domain.Result, linterName string) string {
	return reportingdomain.Diagnostic(result, linterName)
}
