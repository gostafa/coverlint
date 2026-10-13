// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"errors"
)

var (
	errCollectorNotConfigured = errors.New(
		"coverage collector is not configured",
	)
	_ loadRunner = (*Collector)(nil)
)
