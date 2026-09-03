// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package main

import (
	"errors"
	"os"
	"testing"
)

const (
	testZero    = 0
	testOne     = 1
	testSeven   = 7
	flagVersion = "--version"
)

var errBoom = errors.New("boom")

func TestMainDelegatesToCLI(t *testing.T) {
	var (
		gotArgs []string
		gotCode int
	)

	runtime := mainRuntime{
		run: func(args []string) int {
			gotArgs = append([]string(nil), args...)

			return testSeven
		},
		exit: func(code int) { gotCode = code },
	}

	startRuntime(runtime, []string{flagVersion})

	if len(gotArgs) != testOne || gotArgs[testZero] != flagVersion {
		t.Fatalf("args = %v", gotArgs)
	}

	if gotCode != testSeven {
		t.Fatalf("exit code = %d, want %d", gotCode, testSeven)
	}
}

func TestDefaultRuntime(t *testing.T) {
	runtime := defaultRuntime()

	if runtime.run == nil {
		t.Fatal("defaultRuntime().run is nil")
	}

	if runtime.exit == nil {
		t.Fatal("defaultRuntime().exit is nil")
	}
}

func TestMainUsesRuntimeProvider(t *testing.T) {
	previous := errMainRuntime
	previousArgs := os.Args

	t.Cleanup(func() {
		errMainRuntime = previous
		os.Args = previousArgs
	})

	var (
		gotArgs []string
		gotCode int
	)

	errMainRuntime = mainRuntimeError(func() mainRuntime {
		return mainRuntime{
			run: func(args []string) int {
				gotArgs = append([]string(nil), args...)

				return testSeven
			},
			exit: func(code int) { gotCode = code },
		}
	})

	os.Args = []string{"coverlint", flagVersion}

	main()

	if len(gotArgs) != testOne || gotArgs[testZero] != flagVersion {
		t.Fatalf("args = %v", gotArgs)
	}

	if gotCode != testSeven {
		t.Fatalf("exit code = %d, want %d", gotCode, testSeven)
	}
}

func TestMainRuntimeFromFallback(t *testing.T) {
	t.Parallel()

	runtime := mainRuntimeFrom(errBoom)
	if runtime.run == nil || runtime.exit == nil {
		t.Fatal("fallback runtime missing run/exit")
	}

	var nilProvider mainRuntimeError

	runtime = mainRuntimeFrom(nilProvider)
	if runtime.run == nil || runtime.exit == nil {
		t.Fatal("nil provider missing run/exit")
	}
}

func TestMainRuntimeErrorMethods(t *testing.T) {
	t.Parallel()

	provider := mainRuntimeError(defaultRuntime)

	if provider.Error() != mainRuntimeErrorName || provider.Name() != mainRuntimeErrorName {
		t.Fatalf("Error/Name = %q/%q", provider.Error(), provider.Name())
	}

	if provider.Unwrap() != nil {
		t.Fatal("Unwrap() want nil")
	}

	if provider.Runtime().run == nil {
		t.Fatal("Runtime().run is nil")
	}
}
