// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"errors"
	"path/filepath"
)

var (
	errNonPositiveTimeout = errors.New("timeout must be greater than zero")
	errManagedTestFlag    = errors.New("test argument overrides a coverlint-managed flag")
	errUnknownConfigField = errors.New("unknown coverage config field")
	errAmbiguousTestArgs  = errors.New("ambiguous coverage config")

	// version is the injectable [filepath.Abs] used as absolutePath in resolveResultPath.
	// The identifier is required by gochecknoglobals' allowlist (same as err* globals).
	version = filepath.Abs

	_ jsonObject = decoderFunc(nil)
)
