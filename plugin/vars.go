// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package plugin

import (
	"errors"

	"github.com/golangci/plugin-module-register/register"
)

var (
	// ErrInvalidSettings identifies unsupported plugin configuration data.
	ErrInvalidSettings = errors.New("invalid plugin settings")

	_ register.LinterPlugin = Plugin(nil)

	_ = registerCoverlint()
)
