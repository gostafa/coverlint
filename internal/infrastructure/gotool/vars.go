// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"errors"
	"os"
)

var (
	errGoTestFailed          = errors.New("go test failed")
	errGoListFailed          = errors.New("go list failed")
	errEmptyCoverageProfile  = errors.New(errEmptyCoverageProfileMsg)
	errMissingProfileMode    = errors.New("coverage profile has no mode header")
	errMalformedProfileLine  = errors.New("coverage profile line is malformed")
	errInvalidStatementCount = errors.New("coverage profile line has invalid statement count")
	errInvalidExecutionCount = errors.New("coverage profile line has invalid execution count")

	closeFile = func(file *os.File) error {
		return file.Close()
	}

	_ Toolchain  = (*Adapter)(nil)
	_ BufferView = (*CappedBuffer)(nil)
)
