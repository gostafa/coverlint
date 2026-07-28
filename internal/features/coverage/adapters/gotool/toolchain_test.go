package gotool_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/adapters/gotool"
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
)

func TestCollectRunsGoTestAndParsesProfile(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	coverage, err := gotool.New().Collect(
		context.Background(),
		ports.CoverageRequest{Patterns: []string{dir}, TestArgs: []string{"-run", "TestAdd"}},
	)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if !strings.HasPrefix(string(coverage.Profile), "mode: atomic\n") {
		t.Fatalf("Profile = %q, want atomic mode", coverage.Profile)
	}

	if len(coverage.Blocks) == 0 {
		t.Fatal("Blocks is empty")
	}
}

func TestCollectWrapsGoTestFailureOutput(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	_, err := gotool.New().Collect(
		context.Background(),
		ports.CoverageRequest{Patterns: []string{dir}, TestArgs: []string{"-run", "TestFail"}},
	)
	if err == nil || !strings.Contains(err.Error(), "go test failed") ||
		!strings.Contains(err.Error(), "intentional failure") {
		t.Fatalf("error = %v, want go test output", err)
	}
}

func TestCollectReportsContextCancellation(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := gotool.New().
		Collect(ctx, ports.CoverageRequest{Patterns: []string{dir}, TestArgs: nil})
	if err == nil || !strings.Contains(err.Error(), "go test context") {
		t.Fatalf("error = %v, want context wrapper", err)
	}
}

func TestListReturnsPackageMetadata(t *testing.T) {
	t.Parallel()

	dir := writeGoModule(t)

	packages, err := gotool.New().List(
		context.Background(),
		ports.PackageRequest{
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

	if !strings.HasSuffix(packages[0].ImportPath, strings.TrimPrefix(dir, ".")) {
		t.Fatalf("ImportPath = %q", packages[0].ImportPath)
	}

	if len(packages[0].Files) != 1 || !filepath.IsAbs(packages[0].Files[0]) {
		t.Fatalf("Files = %#v, want absolute source file", packages[0].Files)
	}
}

func TestListWrapsGoListFailure(t *testing.T) {
	t.Parallel()

	_, err := gotool.New().List(
		context.Background(),
		ports.PackageRequest{Patterns: []string{"./missing"}, TestArgs: nil},
	)
	if err == nil || !strings.Contains(err.Error(), "go list failed") {
		t.Fatalf("error = %v, want go list wrapper", err)
	}
}

func TestListReportsStartFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := gotool.New().List(ctx, ports.PackageRequest{Patterns: []string{"."}, TestArgs: nil})
	if err == nil || !strings.Contains(err.Error(), "start go list") {
		t.Fatalf("error = %v, want start wrapper", err)
	}
}

func TestOpenRejectsEmptyProfile(t *testing.T) {
	t.Parallel()

	err := gotool.New().Open(context.Background(), nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "coverage profile is empty") {
		t.Fatalf("error = %v, want empty profile error", err)
	}
}

func TestOpenWrapsGoCoverFailure(t *testing.T) {
	t.Parallel()

	err := gotool.New().Open(
		context.Background(),
		[]byte("not a profile\n"),
		io.Discard,
		io.Discard,
	)
	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want go cover wrapper", err)
	}
}

func TestOpenReportsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := gotool.New().Open(ctx, []byte("mode: atomic\n"), io.Discard, io.Discard)
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

	dir := filepath.Join(".", "coverlint-fixture-"+fixtureName(t))

	err := os.Mkdir(dir, 0o700)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

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
	t.Fatal("intentional failure")
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
