// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package resultcache

import (
	"context"
)

type (
	valueProvider[Value any] interface {
		Get(ctx context.Context) (Value, error)
	}
)
