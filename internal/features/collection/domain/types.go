// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package domain defines raw package and test coverage data independently of policy.
package domain

type (
	// Inputs contains raw coverage and the packages it describes.
	Inputs = struct {
		// Packages selects package patterns or describes collected package metadata.
		Packages []Package
		// Coverage contains coverage data or the computed coverage percentage.
		Coverage Coverage
	}

	// Block describes one coverage profile block.
	Block = struct {
		// File locates the diagnostic or coverage block source.
		File string
		// Position identifies the coverage block coordinates.
		Position string
		// Statements counts coverable statements.
		Statements int64
		// Covered records covered statements or whether the block executed.
		Covered bool
	}

	// Coverage contains a raw coverage profile and its parsed blocks.
	Coverage = struct {
		// TestOutput contains current-run test output.
		TestOutput string
		// Profile contains the atomic Go coverprofile.
		Profile []byte
		// Blocks contains parsed coverage blocks.
		Blocks []Block
		// FailedPackages identifies packages with failed tests.
		FailedPackages []string
		// Failures contains captured failed-test details.
		Failures []TestFailure
		// TestsFailed records test execution failures.
		TestsFailed bool
	}

	// Package describes a Go package and its source files.
	Package = struct {
		// ImportPath identifies the Go package.
		ImportPath string
		// Dir is the package source directory.
		Dir string
		// FirstFile is the first package source file for diagnostics.
		FirstFile string
		// Files lists package source files.
		Files []string
	}

	// TestFailure contains captured output for a failed test or package.
	TestFailure = struct {
		// ImportPath identifies the Go package.
		ImportPath string
		// Test identifies the failed test or subtest.
		Test string
		// Output contains the captured failure output.
		Output string
	}
)
