// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/collection/adapters/gotool"
	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
	reportingports "github.com/gostafa/coverlint/internal/features/reporting/ports"
)

var errReadFailed = errors.New("read failed")

func TestParseProfileKeepsBlockPosition(t *testing.T) {
	t.Parallel()

	blocks, err := gotool.ParseProfile(
		strings.NewReader("mode: atomic\nexample.com/app/a.go:10.1,12.2 3 1\n"),
	)
	if err != nil {
		t.Fatalf("parseProfile: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1", len(blocks))
	}

	if blocks[0].Position != "10.1,12.2" {
		t.Fatalf("Position = %q, want profile range", blocks[0].Position)
	}
}

func TestParseProfileSkipsBlankLines(t *testing.T) {
	t.Parallel()

	blocks, err := gotool.ParseProfile(
		strings.NewReader("mode: atomic\n\nexample.com/app/a.go:10.1,12.2 3 0\n"),
	)
	if err != nil {
		t.Fatalf("ParseProfile: %v", err)
	}

	if len(blocks) != 1 || blocks[0].Covered {
		t.Fatalf("blocks = %#v, want one uncovered block", blocks)
	}
}

func TestParseProfileRejectsBadInput(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "coverage profile is empty"},
		{name: "missing mode", input: "not mode\n", want: "coverage profile has no mode header"},
		{name: "missing colon", input: "mode: atomic\nbad\n", want: "coverage profile line is malformed: 2"},
		{name: "bad position", input: "mode: atomic\nfile.go:bad 1 1\n", want: "coverage profile line is malformed: 2"},
		{
			name:  "bad statements",
			input: "mode: atomic\nfile.go:1.1,2.1 nope 1\n",
			want:  "coverage profile line has invalid statement count: 2",
		},
		{
			name:  "negative statements",
			input: "mode: atomic\nfile.go:1.1,2.1 -1 1\n",
			want:  "coverage profile line has invalid statement count: 2",
		},
		{
			name:  "bad count",
			input: "mode: atomic\nfile.go:1.1,2.1 1 nope\n",
			want:  "coverage profile line has invalid execution count: 2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := gotool.ParseProfile(strings.NewReader(test.input))

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestParseProfileWrapsScannerError(t *testing.T) {
	t.Parallel()

	_, err := gotool.ParseProfile(errReader{})

	if err == nil || !strings.Contains(err.Error(), "read coverage profile") {
		t.Fatalf("error = %v, want read wrapper", err)
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errReadFailed
}

func TestCollectRunsGoTestAndParsesProfile(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	coverage, err := goToolForTest().Collect(t.Context(), &ports.CoverageRequest{Patterns: []string{dir}, TestArgs: []string{"-run", "TestAdd"}})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if !strings.HasPrefix(string(coverage.Profile), "mode: atomic\n") {
		t.Fatalf("Profile = %q, want atomic mode", coverage.Profile)
	}

	if len(coverage.Blocks) == 0 {
		t.Fatal("Blocks is empty")
	}

	if coverage.TestOutput == "" {
		t.Fatal("TestOutput is empty on successful Collect")
	}
}

func TestCollectWrapsGoTestFailureOutput(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	coverage, err := goToolForTest().Collect(t.Context(), &ports.CoverageRequest{Patterns: []string{dir}, TestArgs: []string{"-run", "TestFail"}})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	assertSoftFailedCoverage(t, coverage)

	if len(coverage.Failures) != 3 || coverage.Failures[0].Test != "TestFail/case" {
		t.Fatalf("Failures = %#v, want subtest, parent, and package failures", coverage.Failures)
	}

	if !strings.Contains(coverage.Failures[0].Output, "input=2 got=3 want=4") {
		t.Fatalf("failure output = %q", coverage.Failures[0].Output)
	}
}

func assertSoftFailedCoverage(t *testing.T, coverage domain.Coverage) {
	t.Helper()

	if coverage.TestOutput == "" || !strings.Contains(coverage.TestOutput, "intentional failure") {
		t.Fatalf("TestOutput = %q, want intentional failure", coverage.TestOutput)
	}

	if len(coverage.FailedPackages) == 0 {
		t.Fatal("FailedPackages is empty")
	}

	if !strings.HasPrefix(string(coverage.Profile), "mode: atomic\n") {
		t.Fatalf("Profile = %q, want atomic mode", coverage.Profile)
	}

	if len(coverage.Blocks) == 0 {
		t.Fatal("Blocks is empty")
	}
}

func TestCollectReportsContextCancellation(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := goToolForTest().
		Collect(ctx, &ports.CoverageRequest{Patterns: []string{dir}, TestArgs: nil})

	if err == nil || !strings.Contains(err.Error(), "go test context") {
		t.Fatalf("error = %v, want context wrapper", err)
	}
}

func TestListReturnsPackageMetadata(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	packages, err := goToolForTest().List(t.Context(), &ports.PackageRequest{
		Patterns: []string{dir},
		TestArgs: []string{"-tags=unit", "-run=TestAdd"},
	},
	)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(packages) != 1 {
		t.Fatalf("len(packages) = %d, want 1", len(packages))
	}

	assertListedPackage(t, packages[0], dir)
}

func assertListedPackage(t *testing.T, pkg domain.Package, dir string) {
	t.Helper()

	if !strings.HasSuffix(pkg.ImportPath, strings.TrimPrefix(dir, ".")) {
		t.Fatalf("ImportPath = %q", pkg.ImportPath)
	}

	if len(pkg.Files) != 1 {
		t.Fatalf("Files = %#v, want one source file", pkg.Files)
	}

	if !filepath.IsAbs(pkg.Files[0]) {
		t.Fatalf("Files = %#v, want absolute source file", pkg.Files)
	}
}

func goToolForTest() *gotool.Adapter {
	return gotool.New()
}

func TestListWrapsGoListFailure(t *testing.T) {
	t.Parallel()

	_, err := goToolForTest().List(t.Context(), &ports.PackageRequest{Patterns: []string{"./missing"}, TestArgs: nil})

	if err == nil || !strings.Contains(err.Error(), "go list failed") {
		t.Fatalf("error = %v, want go list wrapper", err)
	}
}

func TestListReportsStartFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := goToolForTest().List(ctx, &ports.PackageRequest{Patterns: []string{"."}, TestArgs: nil})

	if err == nil || !strings.Contains(err.Error(), "start go list") {
		t.Fatalf("error = %v, want start wrapper", err)
	}
}

func TestOpenRejectsEmptyProfile(t *testing.T) {
	t.Parallel()

	err := goToolForTest().Open(t.Context(), &reportingports.HTMLOpenRequest{Profile: nil, Stdout: nil, Stderr: nil})

	if err == nil || !strings.Contains(err.Error(), "coverage profile is empty") {
		t.Fatalf("error = %v, want empty profile error", err)
	}
}

func TestOpenWrapsGoCoverFailure(t *testing.T) {
	t.Parallel()

	err := goToolForTest().Open(
		t.Context(),
		&reportingports.HTMLOpenRequest{
			Profile: []byte("not a profile\n"),
			Stdout:  io.Discard,
			Stderr:  io.Discard,
		},
	)

	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want go cover wrapper", err)
	}
}

func TestOpenReportsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := gotool.New().Open(ctx, &reportingports.HTMLOpenRequest{
		Profile: []byte("mode: atomic\n"),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})

	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}

func TestListArgsForTestArgsForwardsBuildContextOnly(t *testing.T) {
	t.Parallel()

	got := gotool.ListArgsForTestArgs([]string{
		"-race",
		"-tags=integration",
		"-overlay", "overlay.json",
		"-modfile=tools.mod",
		"-run=TestUnit",
		"-short",
		"-gcflags", "all=-N -l",
	})

	want := []string{
		"-race",
		"-tags=integration",
		"-overlay", "overlay.json",
		"-modfile=tools.mod",
		"-gcflags", "all=-N -l",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listArgsForTestArgs() = %#v, want %#v", got, want)
	}
}

func TestCappedBufferLimitsStoredOutput(t *testing.T) {
	t.Parallel()

	buffer := gotool.NewCappedBuffer(5)

	_, err := buffer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := buffer.String(); got != "hello\n... output truncated by coverlint ..." {
		t.Fatalf("String() = %q", got)
	}
}

func TestCappedBufferHandlesZeroLimit(t *testing.T) {
	t.Parallel()

	buffer := gotool.NewCappedBuffer(0)

	written, err := buffer.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if written != 5 {
		t.Fatalf("Write count = %d, want 5", written)
	}

	if got := string(buffer.Bytes()); got != "\n... output truncated by coverlint ..." {
		t.Fatalf("Bytes() = %q", got)
	}
}

func TestCappedBufferStopsWritingAfterLimit(t *testing.T) {
	t.Parallel()

	buffer := gotool.NewCappedBuffer(3)

	_, err := buffer.Write([]byte("abc"))
	if err != nil {
		t.Fatalf("first Write: %v", err)
	}

	_, err = buffer.Write([]byte("def"))
	if err != nil {
		t.Fatalf("second Write: %v", err)
	}

	if got := buffer.String(); got != "abc\n... output truncated by coverlint ..." {
		t.Fatalf("String() = %q", got)
	}
}

func writeGoModule(t *testing.T) string {
	t.Helper()

	dir := moduleFixtureDir(t)

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	writeFixtureFile(
		t,
		dir,
		"calc.go",
		"package fixture\n\nfunc Add(a, b int) int { return a + b }\n",
	)
	writeFixtureFile(t, dir, "calc_test.go", `package fixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}

func TestFail(t *testing.T) {
	t.Run("case", func(t *testing.T) {
		t.Fatal("intentional failure: input=2 got=3 want=4")
	})
}
`)

	return "./" + filepath.Base(dir)
}

func writeFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	if err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

func fixtureName(t *testing.T) string {
	t.Helper()

	return strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
}

func moduleFixtureDir(t *testing.T) string {
	t.Helper()

	temp := t.TempDir()
	suffix := filepath.Base(filepath.Dir(temp)) + "-" + filepath.Base(temp)
	dir := filepath.Join(".", "coverlint-fixture-"+fixtureName(t)+"-"+suffix)

	err := os.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	return dir
}
