// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverage

import (
	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

type (
	// Rule is a package coverage rule.
	Rule = domain.Rule
	// Config contains user-provided coverage settings.
	Config = config.Config
	// Result describes one package's coverage policy outcome.
	Result = domain.Result
	// Report summarizes coverage policy outcomes.
	Report = domain.Report
	// Run contains a completed coverage check and deferred report actions.
	Run = coverlint.Run
)
