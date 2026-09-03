// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/application"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

const repoPkgFile = "/repo/pkg/a.go"

var errBoom = errors.New("boom")

func TestCheckerEvaluatesPolicy(t *testing.T) {
	t.Parallel()

	coverage, packages := checkerPortsForTest()

	policy, err := domain.NewPolicy([]domain.Rule{{Pattern: "**", Min: 0.90}}, nil)
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	outcome, err := application.NewChecker(coverage, packages).Check(
		t.Context(),
		&application.Request{
			Policy:   policy,
			Patterns: []string{"./pkg"},
			Timeout:  time.Minute,
			TestArgs: []string{"-race"},
		},
	)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	assertCheckerOutcome(t, outcome)
	assertCheckerRequests(t, coverage.requests, packages.requests)
}

func checkerPortsForTest() (*fakeCoverageRunner, *fakePackageCatalog) {
	return &fakeCoverageRunner{
		result: domain.Coverage{
			Profile: []byte("mode: atomic\n"),
			Blocks: []domain.Block{{
				File:       repoPkgFile,
				Position:   "",
				Statements: 10,
				Covered:    true,
			}},
		},
		err:      nil,
		requests: nil,
	}, &fakePackageCatalog{
		result: []domain.Package{{
			ImportPath: "example.com/repo/pkg",
			Dir:        "",
			Files:      []string{repoPkgFile},
			FirstFile:  repoPkgFile,
		}},
		err:      nil,
		requests: nil,
	}
}

func assertCheckerOutcome(t *testing.T, outcome application.Outcome) {
	t.Helper()

	if string(outcome.Profile) != "mode: atomic\n" {
		t.Fatalf("Profile = %q", outcome.Profile)
	}

	if outcome.Report.Checked != 1 || outcome.Report.Failed != 0 {
		t.Fatalf("Report = %#v", outcome.Report)
	}
}

func assertCheckerRequests(
	t *testing.T,
	coverageRequests []outbound.CoverageRequest,
	packageRequests []outbound.PackageRequest,
) {
	t.Helper()

	if len(coverageRequests) != 1 || coverageRequests[0].Patterns[0] != "./pkg" {
		t.Fatalf("coverage requests = %#v", coverageRequests)
	}

	if len(packageRequests) != 1 || packageRequests[0].TestArgs[0] != "-race" {
		t.Fatalf("package requests = %#v", packageRequests)
	}
}

func TestCheckerRequiresPorts(t *testing.T) {
	t.Parallel()

	_, err := application.NewChecker(nil, nil).Check(
		t.Context(),
		requestForTest(time.Duration(0)),
	)
	if err == nil {
		t.Fatal("Check succeeded, want configuration error")
	}
}

func TestCheckerWrapsCollectError(t *testing.T) {
	t.Parallel()

	checker := application.NewChecker(
		&fakeCoverageRunner{
			result:   domain.Coverage{Profile: nil, Blocks: nil},
			err:      errBoom,
			requests: nil,
		},
		&fakePackageCatalog{result: nil, err: nil, requests: nil},
	)

	_, err := checker.Check(t.Context(), requestForTest(time.Minute))

	if err == nil || !strings.Contains(err.Error(), "collect coverage: boom") {
		t.Fatalf("error = %v, want collect wrapper", err)
	}
}

func TestCheckerWrapsListError(t *testing.T) {
	t.Parallel()

	checker := application.NewChecker(
		&fakeCoverageRunner{
			result:   domain.Coverage{Profile: nil, Blocks: nil},
			err:      nil,
			requests: nil,
		},
		&fakePackageCatalog{result: nil, err: errBoom, requests: nil},
	)

	_, err := checker.Check(t.Context(), requestForTest(time.Minute))

	if err == nil || !strings.Contains(err.Error(), "list packages: boom") {
		t.Fatalf("error = %v, want list wrapper", err)
	}
}

func TestCheckerReportsTimeout(t *testing.T) {
	t.Parallel()

	checker := application.NewChecker(
		&fakeCoverageRunner{
			result:   domain.Coverage{Profile: nil, Blocks: nil},
			err:      context.DeadlineExceeded,
			requests: nil,
		},
		&fakePackageCatalog{result: nil, err: nil, requests: nil},
	)

	_, err := checker.Check(t.Context(), requestForTest(time.Nanosecond))

	if err == nil || !strings.Contains(err.Error(), "coverage check exceeded timeout") {
		t.Fatalf("error = %v, want timeout wrapper", err)
	}
}

func requestForTest(timeout time.Duration) *application.Request {
	return &application.Request{
		Policy:   domain.Policy{},
		Patterns: nil,
		Timeout:  timeout,
		TestArgs: nil,
	}
}

type fakeCoverageRunner struct {
	result   domain.Coverage
	err      error
	requests []outbound.CoverageRequest
}

func (f *fakeCoverageRunner) Collect(
	_ context.Context,
	request *outbound.CoverageRequest,
) (domain.Coverage, error) {
	f.requests = append(f.requests, *request)

	return f.result, f.err
}

type fakePackageCatalog struct {
	result   []domain.Package
	err      error
	requests []outbound.PackageRequest
}

func (f *fakePackageCatalog) List(
	_ context.Context,
	request *outbound.PackageRequest,
) ([]domain.Package, error) {
	f.requests = append(f.requests, *request)

	return f.result, f.err
}
