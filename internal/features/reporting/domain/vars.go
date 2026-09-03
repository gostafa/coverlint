// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"os"
	"sync"
)

var (
	workingDirectory   = os.Getwd
	workingDirectoryMu sync.RWMutex
)
