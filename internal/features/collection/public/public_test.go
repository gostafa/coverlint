// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package public

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gostafa/coverlint/internal/features/collection/application"
	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
	"github.com/gostafa/coverlint/internal/features/collection/public/inputs"
)

type coverageFunc func(context.Context, *ports.CoverageRequest) (domain.Coverage, error)

func (run coverageFunc) Collect(
	ctx context.Context,
	request *ports.CoverageRequest,
) (domain.Coverage, error) {
	return run(ctx, request)
}

type packageFunc func(context.Context, *ports.PackageRequest) ([]domain.Package, error)

func (list packageFunc) List(
	ctx context.Context,
	request *ports.PackageRequest,
) ([]domain.Package, error) {
	return list(ctx, request)
}

func TestLoadMergesFailuresAndCopiesPublishedData(t *testing.T) {
	t.Parallel()
	coverage := domain.Coverage{
		Profile:    []byte("profile"),
		TestOutput: "current",
		Blocks: []domain.Block{
			{File: "file.go", Position: "1.1,1.2", Statements: 1, Covered: true},
		},
	}
	packages := []domain.Package{
		{ImportPath: "pkg", Dir: "/repo", FirstFile: "file.go", Files: []string{"file.go"}},
	}
	service := &Service{load: collectionLoader(application.NewCollector(
		coverageFunc(
			func(context.Context, *ports.CoverageRequest) (domain.Coverage, error) { return coverage, nil },
		),
		packageFunc(
			func(context.Context, *ports.PackageRequest) ([]domain.Package, error) { return packages, nil },
		),
	))}
	imported := inputs.Coverage{
		TestsFailed:    true,
		FailedPackages: []string{"pkg"},
		Failures: []inputs.TestFailure{
			{ImportPath: "pkg", Test: "Old", Output: "old failure"},
		},
	}
	value, err := service.Load(
		t.Context(),
		&Request{Patterns: []string{"."}, TestArgs: []string{"-race"}, Imported: imported},
	)
	if err != nil || !value.Coverage.TestsFailed || value.Coverage.TestOutput != "current" ||
		len(value.Coverage.Failures) != 1 ||
		len(value.Coverage.Blocks) != 1 {
		t.Fatalf("inputs=%#v err=%v", value, err)
	}
	value.Coverage.Profile[0] = 'x'
	value.Packages[0].Files[0] = "changed"
	value.Coverage.Failures[0].Output = "changed"
	if string(coverage.Profile) != "profile" || packages[0].Files[0] != "file.go" ||
		imported.Failures[0].Output != "old failure" {
		t.Fatal("published data aliases implementation/caller data")
	}
}

func TestLoadTranslatesFailures(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("collector failed")
	service := &Service{
		load: collectionLoader(application.NewCollector(
			coverageFunc(func(context.Context, *ports.CoverageRequest) (domain.Coverage, error) {
				return domain.Coverage{}, sentinel
			}),
			packageFunc(
				func(context.Context, *ports.PackageRequest) ([]domain.Package, error) { return nil, nil },
			),
		)),
	}
	if _, err := service.Load(t.Context(), &Request{}); !errors.Is(err, ports.ErrCollection) {
		t.Fatalf("failure=%v", err)
	}
}

func TestReadSavedReports(t *testing.T) {
	t.Parallel()
	service := New()
	if value, err := service.ReadReport(t.Context(), ""); err != nil || value.TestsFailed {
		t.Fatalf("empty=%#v %v", value, err)
	}
	path := filepath.Join(t.TempDir(), "report")
	if err := os.WriteFile(path, []byte("--- FAIL: TestOld (0.00s)\nFAIL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if value, err := service.ReadReport(
		t.Context(),
		path,
	); err != nil || !value.TestsFailed ||
		len(value.Failures) != 1 {
		t.Fatalf("saved=%#v %v", value, err)
	}
	if _, err := service.ReadReport(t.Context(), t.TempDir()); err == nil {
		t.Fatal("directory read succeeded")
	}
}
