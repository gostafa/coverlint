// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package collection connects policy inputs to the published collection API.
package collection

import (
	"github.com/gostafa/coverlint/internal/features/collection/public"
)

type (
	// Adapter translates collection inputs into policy-owned data.
	Adapter struct{ source public.Collector }
)
