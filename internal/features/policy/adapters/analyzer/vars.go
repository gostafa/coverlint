// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	reportingports "github.com/gostafa/coverlint/internal/features/policy/ports/resultcache"
)

var (
	_ reportingports.ResultProvider = (*snapshotProvider)(nil)

	_ analyzerResult = runResult{}
)
