// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package resultcache

import (
	reportingports "github.com/gostafa/coverlint/internal/features/reporting/ports"
)

var _ reportingports.ResultProvider = (*Adapter)(nil)
