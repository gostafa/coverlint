// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"errors"

	"github.com/gostafa/coverlint/internal/features/reporting/ports"
)

var (
	errEmptyCoverageProfile = errors.New("coverage profile is empty")

	_ ports.HTMLReporter = (*Adapter)(nil)
)
