// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

// Rule sets a minimum coverage percentage for packages matching Pattern.
type Rule struct {
	Pattern string  `json:"pattern"`
	Min     float64 `json:"min"`
}

// Package describes a Go package and its source files.
type Package struct {
	ImportPath string
	Dir        string
	FirstFile  string
	Files      []string
}

// Block describes one coverage profile block.
type Block struct {
	File       string
	Position   string
	Statements int64
	Covered    bool
}

// Coverage contains a raw coverage profile and its parsed blocks.
type Coverage struct {
	Profile []byte
	Blocks  []Block
}

// Result describes one package's coverage policy outcome.
type Result struct {
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

// Report summarizes coverage policy outcomes.
type Report struct {
	Results []Result
	Checked int
	Failed  int
	Skipped int
}
