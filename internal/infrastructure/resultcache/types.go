// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package resultcache shares one immutable result per analyzer instance.
package resultcache

import (
	"context"

	"github.com/gostafa/coverlint/internal/features/policy/domain"
)

type (
	// Adapter supplies a shared result through the reporting result-provider port.
	Adapter struct {
		get func(context.Context) (map[string]domain.Result, error)
	}

	valueProvider[Value any] interface {
		Get(ctx context.Context) (Value, error)
	}
)
