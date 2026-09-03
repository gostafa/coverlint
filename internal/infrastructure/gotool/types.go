// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"context"
	"io"
	"os/exec"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

type (
	// Adapter runs Go toolchain coverage commands.
	Adapter func()

	// Toolchain collects coverage, lists packages, and opens HTML reports.
	Toolchain interface {
		Collect(ctx context.Context, request *outbound.CoverageRequest) (domain.Coverage, error)
		List(ctx context.Context, request *outbound.PackageRequest) ([]domain.Package, error)
		Open(ctx context.Context, request *outbound.HTMLOpenRequest) error
	}

	// BufferView exposes captured toolchain output.
	BufferView interface {
		Bytes() []byte
		String() string
	}

	goListPackage = struct {
		ImportPath  string
		Dir         string
		GoFiles     []string
		CgoFiles    []string
		TestGoFiles []string
	}

	cappedBufferStore interface {
		Write([]byte) (int, error)
		Len() int
		Bytes() []byte
		String() string
	}

	// CappedBuffer stores command output up to a byte limit.
	CappedBuffer struct {
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
		stderr *CappedBuffer
	}

	goTestRun = struct {
		request     *outbound.CoverageRequest
		output      *CappedBuffer
		profilePath string
	}

	waitErrs = struct {
		decode error
		wait   error
	}
)
