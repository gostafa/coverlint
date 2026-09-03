// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package plugin

import (
	"github.com/golangci/plugin-module-register/register"
)

var (
	_ register.LinterPlugin = (*Plugin)(nil)
	_                       = registerCoverlint()
)
