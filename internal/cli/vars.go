// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"errors"
	"io"
)

var (
	errRuleFormat       = errors.New("rule must have the form pattern:min")
	errNonPositiveValue = errors.New("timeout must be greater than zero")

	_ listValue    = (*stringList)(nil)
	_ indexedError = ruleFormatError{}
	_ io.Writer    = stdWriter(nil)
)
