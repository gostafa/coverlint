// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func benchProfile(packages, filesPerPackage, blocksPerFile int) []byte {
	var buf bytes.Buffer

	buf.WriteString("mode: atomic\n")

	for i := range packages {
		for j := range filesPerPackage {
			for k := range blocksPerFile {
				fmt.Fprintf(&buf, "example.com/app/pkg%d/file%d.go:%d.1,%d.2 %d %d\n",
					i, j, k*3+1, k*3+2, k%5+1, (i+j+k)%3)
			}
		}
	}

	return buf.Bytes()
}

func BenchmarkParseProfile(b *testing.B) {
	for _, packages := range []int{10, 100, 500} {
		profile := benchProfile(packages, 8, 32)

		b.Run(strconv.Itoa(packages)+"_packages", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(profile)))

			for b.Loop() {
				blocks, err := ParseProfile(bytes.NewReader(profile))
				if err != nil {
					b.Fatal(err)
				}

				if len(blocks) != packages*8*32 {
					b.Fatalf("len(blocks) = %d", len(blocks))
				}
			}
		})
	}
}

func benchJSONReport(packages, testsPerPackage int) string {
	var buf strings.Builder

	for i := range packages {
		pkg := fmt.Sprintf("example.com/app/pkg%d", i)

		for j := range testsPerPackage {
			test := fmt.Sprintf("TestCase%d", j)
			action := "pass"

			if (i+j)%7 == 0 {
				action = "fail"
			}

			fmt.Fprintf(&buf, `{"Action":"run","Package":%q,"Test":%q}`+"\n", pkg, test)
			fmt.Fprintf(&buf, `{"Action":"output","Package":%q,"Test":%q,"Output":"=== RUN   %s\n"}`+"\n",
				pkg, test, test)
			fmt.Fprintf(&buf, `{"Action":"output","Package":%q,"Test":%q,"Output":"    file.go:12: some log line\n"}`+"\n",
				pkg, test)
			fmt.Fprintf(&buf, `{"Action":%q,"Package":%q,"Test":%q,"Elapsed":0.01}`+"\n",
				action, pkg, test)
		}

		fmt.Fprintf(&buf, `{"Action":"output","Package":%q,"Output":"ok  \t%s\t0.1s\n"}`+"\n", pkg, pkg)
		fmt.Fprintf(&buf, `{"Action":"pass","Package":%q,"Elapsed":0.1}`+"\n", pkg)
	}

	return buf.String()
}

func benchTextReport(packages, testsPerPackage int) string {
	var buf strings.Builder

	for i := range packages {
		pkg := fmt.Sprintf("example.com/app/pkg%d", i)
		failed := false

		for j := range testsPerPackage {
			fmt.Fprintf(&buf, "=== RUN   TestCase%d\n", j)

			if (i+j)%7 == 0 {
				failed = true

				fmt.Fprintf(&buf, "    file.go:12: unexpected value\n--- FAIL: TestCase%d (0.00s)\n", j)

				continue
			}

			fmt.Fprintf(&buf, "--- PASS: TestCase%d (0.00s)\n", j)
		}

		if failed {
			fmt.Fprintf(&buf, "FAIL\nFAIL\t%s\t0.1s\n", pkg)

			continue
		}

		fmt.Fprintf(&buf, "PASS\nok  \t%s\t0.1s\n", pkg)
	}

	return buf.String()
}

func BenchmarkReadTestReport(b *testing.B) {
	dir := b.TempDir()
	reports := []struct {
		name    string
		content string
	}{
		{"json", benchJSONReport(50, 20)},
		{"text", benchTextReport(50, 20)},
	}

	for _, report := range reports {
		path := filepath.Join(dir, report.name+".txt")

		if err := os.WriteFile(path, []byte(report.content), 0o600); err != nil {
			b.Fatal(err)
		}

		b.Run(report.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(report.content)))

			for b.Loop() {
				if _, err := ReadTestReport(path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkListArgsForTestArgs(b *testing.B) {
	args := []string{
		"-race", "-count=1", "-run", "TestFoo", "-tags", "integration,e2e",
		"-timeout=5m", "-v", "-coverpkg=./...", "-shuffle=on", "./...",
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = ListArgsForTestArgs(args)
	}
}
