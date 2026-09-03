// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverage

import (
	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
	reportingdomain "github.com/gostafa/coverlint/internal/features/reporting/domain"
)

const (
	// Name is the golangci-lint plugin and diagnostic category name.
	Name = coverlint.Name
	// DefaultMinimum is the default required package coverage percentage.
	DefaultMinimum = config.DefaultMinimum
)

type (
	// Override is a package-specific coverage rule.
	Override = domain.Rule
	// Config contains user-provided coverage settings.
	Config = config.Config
	// Result describes one package's coverage policy outcome.
	Result = domain.Result
	// Report summarizes coverage policy outcomes.
	Report = domain.Report
	// Run contains a completed coverage check and deferred report actions.
	Run = coverlint.Run
)

// Check delegates to the coverlint facade.
var Check = coverlint.Check

// ValidateMinimum delegates to the coverlint facade.
var ValidateMinimum = coverlint.ValidateMinimum

// HTMLReporter re-exports the port interface.
type HTMLReporter = ports.HTMLReporter

// Diagnostic formats a coverage result as a linter diagnostic line.
func Diagnostic(result Result) string {
	return reportingdomain.Diagnostic(result, Name)
}
