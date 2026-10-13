// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package application

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	policyinputs "github.com/gostafa/coverlint/internal/features/policy/domain/inputs"
	"github.com/gostafa/coverlint/internal/features/policy/ports"
)

type inputFunc func(context.Context, *ports.CoverageRequest) (policyinputs.Inputs, error)

func (load inputFunc) Load(
	ctx context.Context,
	request *ports.CoverageRequest,
) (policyinputs.Inputs, error) {
	return load(ctx, request)
}

type readerFunc func(context.Context, string) (policyinputs.Coverage, error)

func (read readerFunc) Read(ctx context.Context, path string) (policyinputs.Coverage, error) {
	return read(ctx, path)
}

type writerFunc func(context.Context, string, []byte) error

func (write writerFunc) Write(ctx context.Context, path string, data []byte) error {
	return write(ctx, path, data)
}

func TestWorkflowImportsBeforeCollectionAndWritesCurrentArtifactsInOrder(t *testing.T) {
	t.Parallel()
	events := []string{}
	request := &Request{
		Timeout:            time.Minute,
		TestResultPath:     "test.txt",
		CoverageResultPath: "coverage.out",
	}
	collector := inputFunc(
		func(_ context.Context, input *ports.CoverageRequest) (policyinputs.Inputs, error) {
			events = append(events, "collect")
			if !input.Imported.TestsFailed || input.Imported.Failures[0].Test != "Old" {
				t.Fatal("import was not supplied to collector")
			}
			return policyinputs.Inputs{
				Coverage: policyinputs.Coverage{
					TestOutput:  "current output",
					Profile:     []byte("profile"),
					TestsFailed: true,
				},
			}, nil
		},
	)
	deps := Dependencies{
		Reports: readerFunc(func(_ context.Context, path string) (policyinputs.Coverage, error) {
			events = append(events, "read:"+path)
			return policyinputs.Coverage{
				TestsFailed: true,
				Failures:    []policyinputs.TestFailure{{Test: "Old"}},
			}, nil
		}),
		Artifacts: writerFunc(func(_ context.Context, path string, data []byte) error {
			events = append(events, "write:"+path+":"+string(data))
			return nil
		}),
	}
	outcome, err := NewChecker(collector, deps).Check(t.Context(), request)
	if err != nil || outcome.Report.Failed != 1 {
		t.Fatalf("outcome=%#v err=%v", outcome, err)
	}
	want := []string{
		"read:test.txt",
		"collect",
		"write:test.txt:current output",
		"write:coverage.out:profile",
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events=%v want=%v", events, want)
	}
	if request.TestReport.TestsFailed {
		t.Fatal("use case changed caller request")
	}
}

func TestWorkflowStopsAtFailureAndRequiresConfiguredPorts(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"read", "collect", "first write", "second write", "missing reader", "missing writer"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			sentinel := io.ErrShortWrite
			writes := 0
			collector := inputFunc(
				func(context.Context, *ports.CoverageRequest) (policyinputs.Inputs, error) {
					if phase == "collect" {
						return policyinputs.Inputs{}, sentinel
					}
					return policyinputs.Inputs{
						Coverage: policyinputs.Coverage{
							TestOutput: "current",
							Profile:    []byte("profile"),
						},
					}, nil
				},
			)
			deps := Dependencies{
				Reports: readerFunc(func(context.Context, string) (policyinputs.Coverage, error) {
					if phase == "read" {
						return policyinputs.Coverage{}, sentinel
					}
					return policyinputs.Coverage{}, nil
				}),
				Artifacts: writerFunc(func(context.Context, string, []byte) error {
					writes++
					if phase == "first write" || phase == "second write" && writes == 2 {
						return sentinel
					}
					return nil
				}),
			}
			if phase == "missing reader" {
				deps.Reports = nil
			}
			if phase == "missing writer" {
				deps.Artifacts = nil
			}
			_, err := NewChecker(
				collector,
				deps,
			).Check(t.Context(), &Request{Timeout: time.Minute, TestResultPath: "test.txt", CoverageResultPath: "coverage.out"})
			if err == nil {
				t.Fatal("failure ignored")
			}
			if phase == "missing reader" || phase == "missing writer" {
				if !errors.Is(err, errCheckerNotConfigured) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, sentinel) {
				t.Fatal(err)
			}
			expected := 0
			if phase == "first write" {
				expected = 1
			}
			if phase == "second write" {
				expected = 2
			}
			if writes != expected {
				t.Fatalf("writes=%d expected=%d", writes, expected)
			}
		})
	}
}
