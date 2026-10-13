// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
)

type (
	// Evaluation contains raw inputs and deterministic path context.
	Evaluation = struct {
		WorkingDirectory string
		Packages         []inputs.Package
		Blocks           []inputs.Block
	}

	globPattern = struct {
		segments []string
	}

	globMatcher = struct {
		memo         map[[indexPairSize]int]bool
		visited      map[[indexPairSize]int]bool
		segments     []string
		pathSegments []string
	}

	// Report summarizes coverage policy outcomes.
	Report = struct {
		// Results contains individual package outcomes.
		Results []Result
		// Checked counts evaluated packages.
		Checked int
		// Failed counts reported violations.
		Failed int
		// Skipped records packages excluded from evaluation.
		Skipped int
	}

	// Policy evaluates coverage blocks against ordered rules.
	Policy = struct {
		rules []compiledRule
	}

	compiledRule = struct {
		rule Rule
		glob globPattern
	}

	packageStats = struct {
		covered    int64
		statements int64
		blocks     int
	}

	packageIndex = struct {
		files map[string]fileMatch
	}

	fileMatch = struct {
		importPath string
		file       string
	}

	indexFileAdd = struct {
		index    packageIndex
		cwd      string
		pkg      *inputs.Package
		filename string
	}

	relativePaths = struct {
		cwd      string
		absolute string
	}

	importPathExtra = struct {
		index    packageIndex
		absolute string
	}

	// blockKey is [importPath, file\x00position] for unique block merging.
	blockKey = [2]string

	mergedBlock = struct {
		statements int64
		covered    bool
	}

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
		// Pattern matches package import paths.
		Pattern string `json:"pattern"`
		// Min is the minimum required coverage fraction.
		Min float64 `json:"min"`
	}

	// Result describes one package's coverage policy outcome.
	Result = struct {
		// Rule identifies the selected coverage rule, or is nil for synthetic outcomes.
		Rule *Rule
		// ImportPath identifies the Go package.
		ImportPath string
		// File locates the diagnostic or coverage block source.
		File string
		// Message contains the user-facing diagnostic.
		Message string
		// Coverage contains coverage data or the computed coverage percentage.
		Coverage float64
		// Statements counts coverable statements.
		Statements int64
		// Covered records covered statements or whether the block executed.
		Covered int64
		// Skipped records packages excluded from evaluation.
		Skipped bool
		// Violation reports a policy or test failure.
		Violation bool
	}
)
