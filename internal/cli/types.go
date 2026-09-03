// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	errOverrideFormat   = errors.New("override must have the form GLOB=MIN")
	errNonPositiveValue = errors.New("timeout must be greater than zero")
)

type overrideFormatError struct {
	index int
	value string
}

func (e overrideFormatError) Error() string {
	return fmt.Sprintf("override %d %q must have the form GLOB=MIN", e.index, e.value)
}

func (e overrideFormatError) Unwrap() error {
	return errOverrideFormat
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)

	return nil
}

type options struct {
	min         float64
	overrides   stringList
	excludes    stringList
	timeout     time.Duration
	testArgs    stringList
	web         bool
	showVersion bool
}
