// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"errors"
)

var (
	errNonPositiveTimeout = errors.New("timeout must be greater than zero")
	errManagedTestFlag    = errors.New("test argument overrides a coverlint-managed flag")
	errUnknownConfigField = errors.New("unknown coverage config field")
	errAmbiguousTestArgs  = errors.New("ambiguous coverage config")

	_ jsonObject = decoderFunc(nil)
)
