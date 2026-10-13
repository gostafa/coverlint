// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package resultcache defines the analyzer snapshot-provider contract.
package resultcache

import (
	"context"
)

type (
	// Violation contains the published diagnostic data needed by analyzer callbacks.
	Violation = struct {
		ImportPath string
		File       string
		Message    string
	}

	// Snapshot contains immutable policy violations shared by analyzer callbacks.
	Snapshot = struct{ Violations map[string]Violation }

	// ResultProvider supplies a cached snapshot; cancellation ends only the caller's wait.
	ResultProvider interface {
		Get(ctx context.Context) (Snapshot, error)
	}
)
