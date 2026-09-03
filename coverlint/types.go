// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
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
)

// Run contains a completed coverage check and deferred report actions.
type Run struct {
	Report  Report
	profile []byte
	html    ports.HTMLReporter
}

