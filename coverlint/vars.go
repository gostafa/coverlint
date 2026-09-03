// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package coverlint

import (
	"errors"
)

var (
	errHTMLAdapterNotConfigured = errors.New("HTML coverage adapter is not configured")

	_ webOpener = (*Run)(nil)
)
