// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package resultcache shares one immutable value per initialized factory.
package resultcache

import (
	"context"
)

type (
	valueProvider[Value any] interface {
		Get(ctx context.Context) (Value, error)
	}
)
