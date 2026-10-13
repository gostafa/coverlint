// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package collection

import (
	"github.com/gostafa/coverlint/internal/features/policy/ports"
)

var (
	_ ports.InputCollector = (*Adapter)(nil)

	_ ports.ReportReader = (*Adapter)(nil)
)
