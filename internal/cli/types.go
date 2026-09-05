// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"io"
	"time"
)

type (
	listValue interface {
		Set(value string) error
		String() string
		Values() []string
	}

	indexedError interface {
		error
		Index() int
		Value() string
	}

	// ruleFormatError is a malformed coverage rule flag.
	ruleFormatError struct {
		value string
		index int
	}

	// stringList is a repeatable CLI flag value.
	stringList []string

	// stdWriter writes bytes to a process stream.
	stdWriter func([]byte) (int, error)

	options = struct {
		rules       stringList
		testArgs    stringList
		timeout     time.Duration
		web         bool
		showVersion bool
	}

	ioStreams = struct {
		stdout io.Writer
		stderr io.Writer
	}

	ruleParts = struct {
		value       string
		pattern     string
		minimumText string
		index       int
	}
)
