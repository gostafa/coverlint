// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

// Package gotool adapts the Go toolchain for coverage collection and reporting.
package gotool

import (
	"io"
	"os/exec"

	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
)

type (
	// Adapter runs Go toolchain coverage commands.
	Adapter func()

	goListPackage = struct {
		// ImportPath identifies the Go package.
		ImportPath string
		// Dir is the package source directory.
		Dir         string
		GoFiles     []string
		CgoFiles    []string
		TestGoFiles []string
	}

	cappedBufferStore interface {
		Write(data []byte) (n int, err error)
		Len() int
		Bytes() []byte
		String() string
	}

	// cappedBuffer stores command output up to a byte limit.
	cappedBuffer struct {
		buffer    cappedBufferStore
		limit     int
		truncated bool
	}

	cappedWrite = struct {
		buffer    cappedBufferStore
		truncated *bool
		limit     int
	}

	listFlagLookup = struct {
		name     string
		testArgs []string
		index    int
		hasValue bool
	}

	goListRun = struct {
		cmd    *exec.Cmd
		stdout io.Reader
		stderr *cappedBuffer
	}

	testEvent = struct {
		Action  string `json:"action"`
		Package string `json:"package"`
		// Test identifies the failed test or subtest.
		Test string `json:"test"`
		// Output contains the captured failure output.
		Output string `json:"output"`
	}

	testKey = struct {
		pkg  string
		test string
	}

	textTestReport = struct {
		output   *cappedBuffer
		failures []domain.TestFailure
		failed   bool
	}

	testReportParser = struct {
		capture *testCaptureState
		text    textTestReport
		result  domain.Coverage
	}

	testCapture func([]byte) int

	testCaptureState = struct {
		output      *cappedBuffer
		logs        map[testKey][]byte
		packageLogs map[string][]byte
		reportLogs  map[string][]byte
		reportOrder []string
		passed      map[testKey]bool
		order       []testKey
		failed      []testKey
		line        []byte
		drop        bool
	}

	goTestRun = struct {
		capture     *testCaptureState
		request     *ports.CoverageRequest
		output      *cappedBuffer
		profilePath string
	}

	waitErrs = struct {
		decode error
		wait   error
	}
)
