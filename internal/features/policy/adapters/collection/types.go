// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package collection

import (
	"github.com/gostafa/coverlint/internal/features/collection/public"
)

type (
	// Adapter translates collection inputs into policy-owned data.
	Adapter struct{ source public.Collector }
)
