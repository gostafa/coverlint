// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"context"
	"io"

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

	webOpener interface {
		OpenWeb(ctx context.Context, stdout, stderr io.Writer) error
	}

	htmlOpenArgs = struct {
		stdout  io.Writer
		stderr  io.Writer
		profile []byte
	}

	htmlOpener = func(ctx context.Context, args *htmlOpenArgs) error

	// Run contains a completed coverage check and deferred report actions.
	Run struct {
		html    htmlOpener
		profile []byte
		// Report summarizes coverage policy outcomes.
		Report Report
	}
)
