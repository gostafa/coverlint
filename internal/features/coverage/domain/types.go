// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package domain models coverage packages, profiles, and policy evaluation.
package domain

import "github.com/gostafa/coverlint/internal/features/coverage/domain/profile"

type (
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
		Results []Result
		Checked int
		Failed  int
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
		pkg      *Package
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

	// TestFailure contains captured output for a failed test or package.
	TestFailure = profile.TestFailure

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
	Package = profile.Package

	// Block describes one coverage profile block.
	Block = profile.Block

	// Coverage contains a raw coverage profile and its parsed blocks.
	Coverage = profile.Coverage

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
