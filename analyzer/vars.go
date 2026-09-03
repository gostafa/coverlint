// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import "encoding/json"

var (
	_ analyzerResult = runResult{}

	jsonMarshal = json.Marshal
)
