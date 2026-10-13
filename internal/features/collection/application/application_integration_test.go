// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gostafa/coverlint/internal/features/collection/application"
	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
)

func TestCollectorLoadsInputsAndForwardsBuildContext(t *testing.T) {
	t.Parallel()
	request := &ports.CoverageRequest{
		Patterns: []string{"./pkg"},
		TestArgs: []string{"-race"},
	}
	tool := &fakeToolchain{
		coverage: domain.Coverage{Profile: []byte("mode: atomic\n"), TestOutput: "current output"},
		packages: []domain.Package{{ImportPath: "example.com/pkg"}},
	}
	inputs, err := application.NewCollector(tool, tool).Load(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inputs, domain.Inputs{Coverage: tool.coverage, Packages: tool.packages}) {
		t.Fatalf("inputs = %#v", inputs)
	}
	if !reflect.DeepEqual(tool.coverageRequests, []ports.CoverageRequest{*request}) ||
		!reflect.DeepEqual(
			tool.packageRequests,
			[]ports.PackageRequest{
				{Patterns: request.Patterns, TestArgs: request.TestArgs},
			},
		) {
		t.Fatalf("requests = %#v, %#v", tool.coverageRequests, tool.packageRequests)
	}
}

func TestCollectorStopsOnPortErrors(t *testing.T) {
	t.Parallel()
	failure := errors.New("port failed")
	for _, failCoverage := range []bool{true, false} {
		tool := &fakeToolchain{}
		if failCoverage {
			tool.coverageErr = failure
		} else {
			tool.packageErr = failure
		}
		inputs, err := application.NewCollector(tool, tool).
			Load(t.Context(), &ports.CoverageRequest{})
		if !errors.Is(err, failure) || !reflect.DeepEqual(inputs, domain.Inputs{}) {
			t.Fatalf("inputs = %#v, error = %v", inputs, err)
		}
		if failCoverage && len(tool.packageRequests) != 0 {
			t.Fatal("package discovery ran after collection failed")
		}
	}
}

func TestCollectorRequiresBothPorts(t *testing.T) {
	t.Parallel()
	tool := &fakeToolchain{}
	for _, collector := range []*application.Collector{
		{}, application.NewCollector(nil, tool), application.NewCollector(tool, nil),
	} {
		if _, err := collector.Load(t.Context(), &ports.CoverageRequest{}); err == nil {
			t.Fatal("Load succeeded with missing ports")
		}
	}
}

type fakeToolchain struct {
	coverage         domain.Coverage
	packages         []domain.Package
	coverageErr      error
	packageErr       error
	coverageRequests []ports.CoverageRequest
	packageRequests  []ports.PackageRequest
}

func (tool *fakeToolchain) Collect(
	_ context.Context,
	request *ports.CoverageRequest,
) (domain.Coverage, error) {
	tool.coverageRequests = append(tool.coverageRequests, *request)
	return tool.coverage, tool.coverageErr
}

func (tool *fakeToolchain) List(
	_ context.Context,
	request *ports.PackageRequest,
) ([]domain.Package, error) {
	tool.packageRequests = append(tool.packageRequests, *request)
	return tool.packages, tool.packageErr
}
