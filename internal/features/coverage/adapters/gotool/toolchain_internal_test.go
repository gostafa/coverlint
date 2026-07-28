package gotool

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const absGoFile = "/abs/a.go"

func TestPackageFromGoListUsesTestFileWhenNoSourceFiles(t *testing.T) {
	t.Parallel()

	pkg := packageFromGoList(goListPackage{
		ImportPath:  "example.com/fixture",
		Dir:         "/repo/pkg",
		GoFiles:     nil,
		CgoFiles:    nil,
		TestGoFiles: []string{"pkg_test.go"},
	})

	if pkg.FirstFile != filepath.Join("/repo/pkg", "pkg_test.go") {
		t.Fatalf("FirstFile = %q, want test file", pkg.FirstFile)
	}

	if len(pkg.Files) != 0 {
		t.Fatalf("Files = %#v, want no source files", pkg.Files)
	}
}

func TestPackageFromGoListHandlesEmptyPackage(t *testing.T) {
	t.Parallel()

	pkg := packageFromGoList(goListPackage{
		ImportPath:  "example.com/empty",
		Dir:         "/repo/empty",
		GoFiles:     nil,
		CgoFiles:    nil,
		TestGoFiles: nil,
	})
	if pkg.FirstFile != "" {
		t.Fatalf("FirstFile = %q, want empty", pkg.FirstFile)
	}

	if len(pkg.Files) != 0 {
		t.Fatalf("Files = %#v, want empty", pkg.Files)
	}
}

func TestPackageFromGoListKeepsAbsoluteFiles(t *testing.T) {
	t.Parallel()

	pkg := packageFromGoList(goListPackage{
		ImportPath:  "example.com/fixture",
		Dir:         "/repo/pkg",
		GoFiles:     []string{absGoFile},
		CgoFiles:    []string{"cgo.go"},
		TestGoFiles: nil,
	})

	want := []string{absGoFile, filepath.Join("/repo/pkg", "cgo.go")}
	if !reflect.DeepEqual(pkg.Files, want) {
		t.Fatalf("Files = %#v, want %#v", pkg.Files, want)
	}

	if pkg.FirstFile != absGoFile {
		t.Fatalf("FirstFile = %q, want absolute file", pkg.FirstFile)
	}
}

func TestSplitFlagRejectsNonFlags(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"pkg", "-", "--"} {
		if _, _, ok := splitFlag(arg); ok {
			t.Fatalf("splitFlag(%q) ok = true, want false", arg)
		}
	}
}

func TestReadTempProfileWrapsOpenError(t *testing.T) {
	t.Parallel()

	_, err := readTempProfile(filepath.Join(t.TempDir(), "missing.coverprofile"))
	if err == nil || !strings.Contains(err.Error(), "open temporary coverage profile") {
		t.Fatalf("error = %v, want open wrapper", err)
	}
}

func TestReadTempProfileWrapsReadError(t *testing.T) {
	t.Parallel()

	_, err := readTempProfile(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "read temporary coverage profile") {
		t.Fatalf("error = %v, want read wrapper", err)
	}
}
