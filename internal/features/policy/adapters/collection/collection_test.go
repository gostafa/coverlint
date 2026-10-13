// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package collection

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/gostafa/coverlint/internal/features/collection/public"
	collectioninputs "github.com/gostafa/coverlint/internal/features/collection/public/inputs"
	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
	"github.com/gostafa/coverlint/internal/features/policy/ports"
)

type source struct {
	inputs   collectioninputs.Inputs
	imported collectioninputs.Coverage
	failure  error
	request  *public.Request
}

func (service *source) Load(
	_ context.Context,
	request *public.Request,
) (collectioninputs.Inputs, error) {
	service.request = request
	return service.inputs, service.failure
}

func (service *source) ReadReport(context.Context, string) (collectioninputs.Coverage, error) {
	return service.imported, service.failure
}

func TestConnectingAdapterMapsInputsAndImportedFailures(t *testing.T) {
	t.Parallel()
	provider := &source{
		inputs: collectioninputs.Inputs{
			Packages: []collectioninputs.Package{{ImportPath: "pkg", Files: []string{"file.go"}}},
			Coverage: collectioninputs.Coverage{
				Profile: []byte("profile"),
				Blocks:  []collectioninputs.Block{{File: "file.go", Covered: true}},
				Failures: []collectioninputs.TestFailure{
					{ImportPath: "pkg", Test: "TestBad", Output: "details"},
				},
				TestsFailed:    true,
				FailedPackages: []string{"pkg"},
			},
		},
		imported: collectioninputs.Coverage{
			TestsFailed: true,
			Failures:    []collectioninputs.TestFailure{{Test: "TestOld"}},
		},
	}
	adapter := New(provider)
	request := &ports.CoverageRequest{
		Patterns: []string{"."},
		TestArgs: []string{"-race"},
		Imported: policyinputs.Coverage{
			Profile:  []byte("imported"),
			Blocks:   []policyinputs.Block{{File: "old.go"}},
			Failures: []policyinputs.TestFailure{{Test: "TestOld"}},
		},
	}
	value, err := adapter.Load(t.Context(), request)
	if err != nil || value.Packages[0].ImportPath != "pkg" || !value.Coverage.TestsFailed ||
		provider.request.Imported.Failures[0].Test != "TestOld" {
		t.Fatalf("mapped=%#v %v", value, err)
	}
	value.Packages[0].Files[0] = "changed"
	value.Coverage.Profile[0] = 'x'
	provider.request.Imported.Profile[0] = 'x'
	if provider.inputs.Packages[0].Files[0] != "file.go" ||
		string(provider.inputs.Coverage.Profile) != "profile" ||
		string(request.Imported.Profile) != "imported" {
		t.Fatal("mapping aliases mutable data")
	}
	if report, err := adapter.Read(
		t.Context(),
		"saved",
	); err != nil ||
		report.Failures[0].Test != "TestOld" {
		t.Fatalf("report=%#v %v", report, err)
	}
}

func TestConnectingAdapterHidesProviderErrors(t *testing.T) {
	t.Parallel()
	provider := &source{failure: &os.PathError{Op: "read", Path: "fixture", Err: os.ErrPermission}}
	adapter := New(provider)
	_, loadErr := adapter.Load(t.Context(), &ports.CoverageRequest{})
	_, readErr := adapter.Read(t.Context(), "saved")
	for _, err := range []error{loadErr, readErr} {
		var raw *os.PathError
		if !errors.Is(err, ports.ErrCollection) || !errors.Is(err, os.ErrPermission) ||
			errors.As(err, &raw) {
			t.Fatalf("unstable provider failure: %v", err)
		}
	}
}
