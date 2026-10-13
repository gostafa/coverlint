// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package analyzer

import (
	"fmt"

	adapter "github.com/gostafa/coverlint/internal/features/policy/adapters/analyzer"
	"golang.org/x/tools/go/analysis"
)

// New returns a go/analysis Analyzer that eagerly runs the coverage check.
func New(settings *Settings) (*analysis.Analyzer, error) {
	input := settings

	value, err := adapter.New(input)
	if err != nil {
		return nil, fmt.Errorf("New: %w", err)
	}

	return value, nil
}

// UnmarshalSettings accepts snake_case keys plus documented kebab/camel aliases.
func UnmarshalSettings(data []byte, settings *Settings) error {
	err := adapter.UnmarshalSettings(data, settings)
	if err != nil {
		return fmt.Errorf("decode settings: %s: %w", err.Error(), ErrInvalidSettings)
	}

	return nil
}
