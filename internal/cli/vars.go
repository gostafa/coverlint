// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"io"
	"os"
)

var (
	osStdout = func() io.Writer { return os.Stdout }
	osStderr = func() io.Writer { return os.Stderr }
)
