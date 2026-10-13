// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"errors"
)

var (
	errCheckerNotConfigured = errors.New("coverage checker is not configured")

	_ CheckRunner = (*Checker)(nil)
)
