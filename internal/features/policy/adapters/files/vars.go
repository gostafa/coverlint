// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package files

import (
	"github.com/gostafa/coverlint/internal/features/policy/ports"
)

var _ ports.ArtifactWriter = (*Adapter)(nil)
