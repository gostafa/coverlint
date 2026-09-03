// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	infgotool "github.com/gostafa/coverlint/internal/infrastructure/gotool"
)

type (
	// Adapter runs Go toolchain coverage commands.
	Adapter = infgotool.Adapter
	// CappedBuffer stores command output up to a byte limit.
	CappedBuffer = infgotool.CappedBuffer
)
