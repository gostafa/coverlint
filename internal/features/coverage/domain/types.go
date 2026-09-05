// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

type (
	pathMatcher interface {
		Matches(importPath string) bool
	}

	coverageGate interface {
		MinCoverage() float64
	}

	patternSource interface {
		PatternValue() string
	}

	// Rule sets a minimum coverage fraction in [0, 1] for packages matching Pattern.
	Rule struct {
		Pattern string  `json:"pattern"`
		Min     float64 `json:"min"`
	}

	// Package describes a Go package and its source files.
	Package = struct {
		ImportPath string
		Dir        string
		FirstFile  string
		Files      []string
	}

	// Block describes one coverage profile block.
	Block = struct {
		File       string
		Position   string
		Statements int64
		Covered    bool
	}

	// Coverage contains a raw coverage profile and its parsed blocks.
	Coverage = struct {
		TestOutput     string
		Profile        []byte
		Blocks         []Block
		FailedPackages []string
		TestsFailed    bool
	}

	// Result describes one package's coverage policy outcome.
	Result = struct {
		Rule       *Rule
		ImportPath string
		File       string
		Message    string
		Coverage   float64
		Statements int64
		Covered    int64
		Skipped    bool
		Violation  bool
	}
)
