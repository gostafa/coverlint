// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"fmt"
	"strconv"
	"testing"

	collectiondomain "github.com/gostafa/coverlint/internal/features/collection/domain"
)

const (
	benchModule          = "example.com/app"
	benchFilesPerPackage = 8
	benchBlocksPerFile   = 32
)

func benchRules() []Rule {
	return []Rule{
		{Pattern: "**", Min: 0.80},
		{Pattern: "**/internal/**", Min: 0.20},
		{Pattern: benchModule + "/internal/features/*/domain", Min: 0.95},
		{Pattern: benchModule + "/cmd/**", Min: 0.50},
		{Pattern: "**/adapters/*", Min: 0.60},
	}
}

func benchImportPath(index int) string {
	switch index % 4 {
	case 0:
		return fmt.Sprintf("%s/internal/features/f%d/domain", benchModule, index)
	case 1:
		return fmt.Sprintf("%s/internal/features/f%d/adapters/a%d", benchModule, index, index)
	case 2:
		return fmt.Sprintf("%s/cmd/tool%d", benchModule, index)
	default:
		return fmt.Sprintf("%s/pkg/p%d/sub/leaf", benchModule, index)
	}
}

func benchInputs(packageCount int) ([]collectiondomain.Package, []collectiondomain.Block) {
	packages := make([]collectiondomain.Package, 0, packageCount)
	blocks := make([]collectiondomain.Block, 0, packageCount*benchFilesPerPackage*benchBlocksPerFile)

	for i := range packageCount {
		importPath := benchImportPath(i)
		files := make([]string, 0, benchFilesPerPackage)

		for j := range benchFilesPerPackage {
			file := "file" + strconv.Itoa(j) + ".go"
			files = append(files, file)

			for k := range benchBlocksPerFile {
				blocks = append(blocks, collectiondomain.Block{
					File:       importPath + "/" + file,
					Position:   fmt.Sprintf("%d.1,%d.2", k*3+1, k*3+2),
					Statements: int64(k%5 + 1),
					Covered:    (i+j+k)%3 != 0,
				})
			}
		}

		packages = append(packages, collectiondomain.Package{
			ImportPath: importPath,
			Dir:        "/src/" + importPath,
			FirstFile:  "/src/" + importPath + "/" + files[0],
			Files:      files,
		})
	}

	return packages, blocks
}

func BenchmarkEvaluate(b *testing.B) {
	policy, err := NewPolicy(benchRules())
	if err != nil {
		b.Fatal(err)
	}

	for _, packageCount := range []int{10, 100, 500} {
		packages, blocks := benchInputs(packageCount)

		b.Run(strconv.Itoa(packageCount)+"_packages", func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				report := Evaluate(&policy, packages, blocks)
				if len(report.Results) != packageCount {
					b.Fatalf("len(Results) = %d, want %d", len(report.Results), packageCount)
				}
			}
		})
	}
}

func BenchmarkNewPolicy(b *testing.B) {
	rules := benchRules()

	b.ReportAllocs()

	for b.Loop() {
		if _, err := NewPolicy(rules); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatchRule(b *testing.B) {
	policy, err := NewPolicy(benchRules())
	if err != nil {
		b.Fatal(err)
	}

	importPaths := make([]string, 0, 64)
	for i := range cap(importPaths) {
		importPaths = append(importPaths, benchImportPath(i))
	}

	b.ReportAllocs()

	for b.Loop() {
		for _, importPath := range importPaths {
			if matchRule(&policy, importPath) == nil {
				b.Fatalf("no rule for %q", importPath)
			}
		}
	}
}

func BenchmarkRuleMatches(b *testing.B) {
	cases := []struct {
		name       string
		rule       Rule
		importPath string
	}{
		{"double_star", Rule{Pattern: "**", Min: 0.8}, benchModule + "/a/b/c/d/e"},
		{"nested_double_star", Rule{Pattern: "**/internal/**/domain", Min: 0.8}, benchModule + "/x/internal/y/z/domain"},
		{"literal", Rule{Pattern: benchModule + "/cmd/tool", Min: 0.8}, benchModule + "/cmd/tool"},
		{"no_match", Rule{Pattern: "**/adapters/*/x/**/y", Min: 0.8}, benchModule + "/a/adapters/b/c/d/e/f/g"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_ = tc.rule.Matches(tc.importPath)
			}
		})
	}
}
