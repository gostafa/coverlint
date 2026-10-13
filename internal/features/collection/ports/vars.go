// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"errors"
)

var (
	// ErrCollection identifies Go test execution or coverage parsing failures.
	ErrCollection = errors.New("coverage collection failed")

	// ErrPackageList identifies package discovery failures.
	ErrPackageList = errors.New("package listing failed")
)
