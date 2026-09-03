// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"fmt"
	"io"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	infgotool "github.com/gostafa/coverlint/internal/infrastructure/gotool"
)

// New creates a Go toolchain coverage adapter.
func New() *Adapter {
	return infgotool.New()
}

// NewCappedBuffer creates a capped output buffer.
func NewCappedBuffer(limit int) CappedBuffer {
	return infgotool.NewCappedBuffer(limit)
}

// ListArgsForTestArgs returns go list-compatible build flags from go test flags.
func ListArgsForTestArgs(testArgs []string) []string {
	return infgotool.ListArgsForTestArgs(testArgs)
}

// ParseProfile reads Go coverage profile blocks.
func ParseProfile(reader io.Reader) ([]domain.Block, error) {
	blocks, err := infgotool.ParseProfile(reader)
	if err != nil {
		return nil, fmt.Errorf("parse coverage profile: %w", err)
	}

	return blocks, nil
}
