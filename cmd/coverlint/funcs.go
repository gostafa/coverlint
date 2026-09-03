// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package main

import (
	"errors"
	"os"

	"github.com/gostafa/coverlint/internal/cli"
)

func main() {
	startRuntime(mainRuntimeFrom(errMainRuntime), os.Args[1:])
}

func defaultRuntime() mainRuntime {
	return mainRuntime{run: cli.Run, exit: os.Exit}
}

func mainRuntimeFrom(err error) mainRuntime {
	var provider mainRuntimeError

	if !errors.As(err, &provider) {
		return defaultRuntime()
	}

	return provider.Runtime()
}

func startRuntime(runtime mainRuntime, args []string) {
	runtime.exit(runtime.run(args))
}

// Error implements the error interface for main runtime selection.
func (mainRuntimeError) Error() string {
	return mainRuntimeErrorName
}

// Name returns the stable error name for the main runtime provider.
func (provider mainRuntimeError) Name() string {
	return provider.Error()
}

// Runtime returns the injectable main runtime, or the default when unset.
func (provider mainRuntimeError) Runtime() mainRuntime {
	if provider == nil {
		return defaultRuntime()
	}

	return provider()
}

// Unwrap implements error unwrapping for mainRuntimeError.
func (mainRuntimeError) Unwrap() error {
	return nil
}
