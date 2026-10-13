// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package request defines inputs for HTML report actions.
package request

import (
	"io"
)

type (
	// HTMLOpenRequest describes an HTML coverage report open request.
	HTMLOpenRequest = struct {
		Stdout  io.Writer
		Stderr  io.Writer
		Profile []byte
	}
)
