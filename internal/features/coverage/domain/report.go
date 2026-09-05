// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

type (
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
)
