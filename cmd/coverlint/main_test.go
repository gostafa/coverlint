// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package main

import (
	"os"
	"testing"
)

const (
	testZero    = 0
	testOne     = 1
	testSeven   = 7
	flagVersion = "--version"
)

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

	runtime.start([]string{flagVersion})

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
	previousProvider := runtimeProvider
	previousArgs := os.Args

	t.Cleanup(func() {
		runtimeProvider = previousProvider
		os.Args = previousArgs
	})

	var (
		gotArgs []string
		gotCode int
	)

	runtimeProvider = func() mainRuntime {
		return mainRuntime{
			run: func(args []string) int {
				gotArgs = append([]string(nil), args...)

				return testSeven
			},
			exit: func(code int) { gotCode = code },
		}
	}

	os.Args = []string{"coverlint", flagVersion}

	main()

	if len(gotArgs) != testOne || gotArgs[testZero] != flagVersion {
		t.Fatalf("args = %v", gotArgs)
	}

	if gotCode != testSeven {
		t.Fatalf("exit code = %d, want %d", gotCode, testSeven)
	}
}
