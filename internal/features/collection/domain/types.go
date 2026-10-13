// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package domain defines raw package and test coverage data independently of policy.
package domain

type (
	// Inputs contains raw coverage and the packages it describes.
	Inputs = struct {
		Packages []Package
		Coverage Coverage
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
		Failures       []TestFailure
		TestsFailed    bool
	}

	// Package describes a Go package and its source files.
	Package = struct {
		ImportPath string
		Dir        string
		FirstFile  string
		Files      []string
	}

	// TestFailure contains captured output for a failed test or package.
	TestFailure = struct {
		ImportPath string
		Test       string
		Output     string
	}
)
