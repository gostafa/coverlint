// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gostafa/coverlint/internal/features/collection/ports"
	collectionrequest "github.com/gostafa/coverlint/internal/features/collection/ports/request"
	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

const absGoFile = "/abs/a.go"

var (
	errBoom       = errors.New("boom")
	errWriteBoom  = errors.New("write boom")
	errCloseBoom  = errors.New("close boom")
	errDecodeBoom = errors.New("decode boom")
)

type failBuffer struct {
	length int
	err    error
}

func (buf *failBuffer) Write(_ []byte) (int, error) {
	return 0, buf.err
}

func (buf *failBuffer) Len() int {
	return buf.length
}

func (buf *failBuffer) Bytes() []byte {
	return nil
}

func (buf *failBuffer) String() string {
	return emptyString
}

type partialErrReader struct {
	data []byte
	err  error
}

func (reader *partialErrReader) Read(dest []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, reader.err
	}

	written := copy(dest, reader.data)
	reader.data = reader.data[written:]

	return written, nil
}

func TestPackageFromGoListUsesTestFileWhenNoSourceFiles(t *testing.T) {
	t.Parallel()

	pkg := packageFromGoList(&goListPackage{
		ImportPath:  "example.com/fixture",
		Dir:         "/repo/pkg",
		GoFiles:     nil,
		CgoFiles:    nil,
		TestGoFiles: []string{"pkg_test.go"},
	})

	if pkg.FirstFile != filepath.Join("/repo/pkg", "pkg_test.go") {
		t.Fatalf("FirstFile = %q, want test file", pkg.FirstFile)
	}

	if len(pkg.Files) != 0 {
		t.Fatalf("Files = %#v, want no source files", pkg.Files)
	}
}

func TestPackageFromGoListHandlesEmptyPackage(t *testing.T) {
	t.Parallel()

	pkg := packageFromGoList(&goListPackage{
		ImportPath:  "example.com/empty",
		Dir:         "/repo/empty",
		GoFiles:     nil,
		CgoFiles:    nil,
		TestGoFiles: nil,
	})

	if pkg.FirstFile != "" {
		t.Fatalf("FirstFile = %q, want empty", pkg.FirstFile)
	}

	if len(pkg.Files) != 0 {
		t.Fatalf("Files = %#v, want empty", pkg.Files)
	}
}

func TestPackageFromGoListKeepsAbsoluteFiles(t *testing.T) {
	t.Parallel()

	pkg := packageFromGoList(&goListPackage{
		ImportPath:  "example.com/fixture",
		Dir:         "/repo/pkg",
		GoFiles:     []string{absGoFile},
		CgoFiles:    []string{"cgo.go"},
		TestGoFiles: nil,
	})

	want := []string{absGoFile, filepath.Join("/repo/pkg", "cgo.go")}

	if !reflect.DeepEqual(pkg.Files, want) {
		t.Fatalf("Files = %#v, want %#v", pkg.Files, want)
	}

	if pkg.FirstFile != absGoFile {
		t.Fatalf("FirstFile = %q, want absolute file", pkg.FirstFile)
	}
}

func TestSplitFlagRejectsNonFlags(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"pkg", "-", "--"} {
		name, hasValue, ok := splitFlag(arg)
		if ok {
			t.Fatalf(
				"splitFlag(%q) name = %q hasValue = %v ok = true, want false",
				arg,
				name,
				hasValue,
			)
		}
	}
}

func TestReadTempProfileWrapsOpenError(t *testing.T) {
	t.Parallel()

	_, err := readTempProfileContext(
		t.Context(),
		filepath.Join(t.TempDir(), "missing.coverprofile"),
	)

	if err == nil || !strings.Contains(err.Error(), "open temporary coverage profile") {
		t.Fatalf("error = %v, want open wrapper", err)
	}
}

func TestReadTempProfileWrapsReadError(t *testing.T) {
	t.Parallel()

	_, err := readTempProfileContext(t.Context(), t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "read temporary coverage profile") {
		t.Fatalf("error = %v, want read wrapper", err)
	}
}

func TestWriteCappedPropagatesWriteErrors(t *testing.T) {
	t.Parallel()

	truncated := false
	_, err := writeCapped(&cappedWrite{
		buffer:    &failBuffer{err: errWriteBoom},
		truncated: &truncated,
		limit:     8,
	}, []byte("payload"))

	if err == nil || !strings.Contains(err.Error(), "write capped buffer") {
		t.Fatalf("error = %v, want capped write wrapper", err)
	}
}

func TestWriteCappedOverflowPropagatesWriteErrors(t *testing.T) {
	t.Parallel()

	truncated := false
	_, err := writeCappedOverflow(&cappedWrite{
		buffer:    &failBuffer{err: errWriteBoom},
		truncated: &truncated,
		limit:     4,
	}, []byte("payload"), 4)

	if err == nil || !strings.Contains(err.Error(), "write capped buffer limit") {
		t.Fatalf("error = %v, want overflow wrapper", err)
	}
}

func TestWriteCappedAllPropagatesWriteErrors(t *testing.T) {
	t.Parallel()

	truncated := false
	_, err := writeCappedAll(&cappedWrite{
		buffer:    &failBuffer{err: errWriteBoom},
		truncated: &truncated,
		limit:     32,
	}, []byte("payload"))

	if err == nil || !strings.Contains(err.Error(), "write capped buffer data") {
		t.Fatalf("error = %v, want all-write wrapper", err)
	}
}

func TestWriteCappedRemainingPropagatesOverflowErrors(t *testing.T) {
	t.Parallel()

	truncated := false
	_, err := writeCappedRemaining(&cappedWrite{
		buffer:    &failBuffer{length: 1, err: errWriteBoom},
		truncated: &truncated,
		limit:     8,
	}, []byte("payload"))

	if err == nil || !errors.Is(err, errWriteBoom) {
		t.Fatalf("error = %v, want write boom", err)
	}
}

func TestCappedBufferWritePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	truncated := false
	written := writeCappedBytes(&cappedWrite{
		buffer:    &failBuffer{err: errWriteBoom},
		truncated: &truncated,
		limit:     8,
	}, []byte("payload"))
	if written != zero {
		t.Fatalf("written = %d, want 0 on store failure", written)
	}
}

func TestWriteCappedPropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	truncated := false
	_, err := writeCapped(&cappedWrite{
		buffer:    &failBuffer{err: errWriteBoom},
		truncated: &truncated,
		limit:     8,
	}, []byte("payload"))
	if err == nil || !errors.Is(err, errWriteBoom) {
		t.Fatalf("error = %v, want write boom", err)
	}
}

func TestCloseHTMLInputHelpers(t *testing.T) {
	t.Parallel()

	file := createClosedTempFile(t)

	if err := closeHTMLInput(file); err == nil {
		t.Fatal("closeHTMLInput() error = nil, want close failure")
	}

	openFile := createTempFile(t)
	if err := closeHTMLInput(openFile); err != nil {
		t.Fatalf("closeHTMLInput() open file: %v", err)
	}
}

func TestCloseHTMLInputErrBranches(t *testing.T) {
	t.Parallel()

	if err := closeHTMLInputErr(createTempFile(t)); err != nil {
		t.Fatalf("closeHTMLInputErr() success path: %v", err)
	}

	if err := closeHTMLInputErr(createClosedTempFile(t)); err == nil {
		t.Fatal("closeHTMLInputErr() error = nil, want close failure")
	}
}

func TestCloseTempProfileErrorJoinsRemove(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.coverprofile")
	err := closeTempProfileErrorContext(t.Context(), missing, errBoom)

	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want joined boom", err)
	}

	if !strings.Contains(err.Error(), "remove temporary file") {
		t.Fatalf("error = %v, want remove failure", err)
	}
}

func TestCloseCreatedTempProfileCloseFailure(t *testing.T) {
	t.Parallel()

	file := createClosedTempFile(t)
	_, err := closeCreatedTempProfileWithContext(t.Context(), file, closeFile)

	if err == nil || !strings.Contains(err.Error(), "close created temp profile") {
		t.Fatalf("error = %v, want close wrapper", err)
	}
}

func TestJoinGoListWaitErrorBranches(t *testing.T) {
	t.Parallel()

	onlyDecode := joinGoListWaitError(errDecodeBoom, nil)
	if onlyDecode == nil || !errors.Is(onlyDecode, errDecodeBoom) {
		t.Fatalf("error = %v, want decode-only join", onlyDecode)
	}

	both := joinGoListWaitError(errDecodeBoom, errBoom)
	if both == nil || !errors.Is(both, errDecodeBoom) || !errors.Is(both, errBoom) {
		t.Fatalf("error = %v, want joined decode and wait", both)
	}
}

func TestJoinWriteCloseError(t *testing.T) {
	t.Parallel()

	err := joinWriteCloseError(createTempFile(t), 3, errWriteBoom)
	if err == nil || !errors.Is(err, errWriteBoom) {
		t.Fatalf("error = %v, want write boom", err)
	}

	if !strings.Contains(err.Error(), "wrote 3") {
		t.Fatalf("error = %v, want written count", err)
	}
}

func TestJoinProfileReadCloseOnly(t *testing.T) {
	t.Parallel()

	_, err := joinProfileRead([]byte("x"), nil, errCloseBoom)
	if err == nil || !errors.Is(err, errCloseBoom) {
		t.Fatalf("error = %v, want close boom", err)
	}
}

func TestWrapCloseErr(t *testing.T) {
	t.Parallel()

	if err := wrapCloseErr(nil); err != nil {
		t.Fatalf("wrapCloseErr(nil) = %v", err)
	}

	err := wrapCloseErr(errCloseBoom)
	if err == nil || !errors.Is(err, errCloseBoom) {
		t.Fatalf("error = %v, want close boom", err)
	}
}

func TestRemoveFileWrapsError(t *testing.T) {
	t.Parallel()

	err := removeFileContext(t.Context(), filepath.Join(t.TempDir(), "missing"))
	if err == nil || !strings.Contains(err.Error(), "remove temporary file") {
		t.Fatalf("error = %v, want remove wrapper", err)
	}
}

func TestGoListErrorBranches(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := goListError(ctx, errBoom, nil)
	if err == nil || !strings.Contains(err.Error(), "go list context") {
		t.Fatalf("error = %v, want context wrapper", err)
	}

	err = goListError(t.Context(), errBoom, nil)
	if err == nil || !errors.Is(err, errGoListFailed) || !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want empty-output join", err)
	}
}

func TestGoTestErrorEmptyOutput(t *testing.T) {
	t.Parallel()

	err := goTestError(t.Context(), errBoom, "  \n")
	if err == nil || !errors.Is(err, errGoTestFailed) || !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want empty-output join", err)
	}
}

func TestParseFailedPackages(t *testing.T) {
	t.Parallel()

	got := parseFailedPackages(
		"--- FAIL: TestX (0.00s)\nFAIL\nFAIL\t\nFAIL\t [build failed]\nFAIL\texample.com/a\t0.01s\nFAIL\texample.com/b [build failed]\nFAIL\texample.com/a\t0.02s\n",
	)
	want := []string{"example.com/a", "example.com/b"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFailedPackages() = %#v, want %#v", got, want)
	}
}

func TestCollectFailedGoTestHardFails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.coverprofile")
	run := failedGoTestRun(missing, "FAIL\texample.com/pkg\t0.01s\n")

	_, err := collectFailedGoTest(t.Context(), run, errBoom)
	if err == nil || !errors.Is(err, errGoTestFailed) {
		t.Fatalf("error = %v, want go test failed", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = collectFailedGoTest(ctx, failedGoTestRun(missing, "output"), errBoom)
	if err == nil || !strings.Contains(err.Error(), "go test context") {
		t.Fatalf("error = %v, want context wrapper", err)
	}
}

func TestCollectFailedGoTestSoftFailsWithUsableProfile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ok.coverprofile")
	if err := os.WriteFile(path, []byte("mode: atomic\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	coverage, err := collectFailedGoTest(
		t.Context(),
		failedGoTestRun(path, "FAIL\texample.com/pkg\t0.01s\n"),
		errBoom,
	)
	if err != nil {
		t.Fatalf("collectFailedGoTest: %v", err)
	}

	if len(coverage.FailedPackages) != 1 || coverage.FailedPackages[0] != "example.com/pkg" {
		t.Fatalf("FailedPackages = %#v", coverage.FailedPackages)
	}

	if !strings.Contains(coverage.TestOutput, "example.com/pkg") {
		t.Fatalf("TestOutput = %q", coverage.TestOutput)
	}
}

func TestCollectGoTestResultBranches(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ok.coverprofile")
	if err := os.WriteFile(path, []byte("mode: atomic\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	coverage, err := collectGoTestResult(
		t.Context(),
		failedGoTestRun(path, "FAIL\texample.com/pkg\t0.01s\n"),
		errBoom,
	)
	if err != nil {
		t.Fatalf("collectGoTestResult: %v", err)
	}

	if len(coverage.FailedPackages) != 1 || coverage.FailedPackages[0] != "example.com/pkg" {
		t.Fatalf("FailedPackages = %#v", coverage.FailedPackages)
	}

	missing := filepath.Join(t.TempDir(), "missing.coverprofile")
	_, err = collectGoTestResult(
		t.Context(),
		failedGoTestRun(missing, "FAIL\texample.com/pkg\t0.01s\n"),
		errBoom,
	)
	if err == nil || !errors.Is(err, errGoTestFailed) {
		t.Fatalf("error = %v, want go test failed", err)
	}
}

func TestGoListRawMessageExactKey(t *testing.T) {
	t.Parallel()

	raw := map[string]json.RawMessage{jsonKeyImportPath: json.RawMessage(`"example.com/pkg"`)}
	value, ok := goListRawMessage(raw, jsonKeyImportPath)

	if !ok || string(value) != `"example.com/pkg"` {
		t.Fatalf("goListRawMessage() = %s ok=%v", value, ok)
	}
}

func TestUnmarshalGoListStringBranches(t *testing.T) {
	t.Parallel()

	var target string

	if err := unmarshalGoListString(map[string]json.RawMessage{}, jsonKeyDir, &target); err != nil {
		t.Fatalf("missing key: %v", err)
	}

	if target != "" {
		t.Fatalf("target = %q, want empty", target)
	}

	err := unmarshalGoListString(
		map[string]json.RawMessage{jsonKeyDir: json.RawMessage(`123`)},
		jsonKeyDir,
		&target,
	)
	if err == nil || !strings.Contains(err.Error(), "unmarshal dir") {
		t.Fatalf("error = %v, want bad JSON wrapper", err)
	}
}

func TestUnmarshalGoListStringsBadJSON(t *testing.T) {
	t.Parallel()

	var target []string

	err := unmarshalGoListStrings(
		map[string]json.RawMessage{jsonKeyGoFiles: json.RawMessage(`"nope"`)},
		jsonKeyGoFiles,
		&target,
	)
	if err == nil || !strings.Contains(err.Error(), "unmarshal goFiles") {
		t.Fatalf("error = %v, want bad JSON wrapper", err)
	}
}

func TestAssignGoListPackageFieldError(t *testing.T) {
	t.Parallel()

	_, err := assignGoListPackage(map[string]json.RawMessage{
		jsonKeyImportPath: json.RawMessage(`false`),
	})
	if err == nil || !strings.Contains(err.Error(), "unmarshal go list package") {
		t.Fatalf("error = %v, want assign wrapper", err)
	}
}

func TestRunGoListAssignsStopsOnError(t *testing.T) {
	t.Parallel()

	err := runGoListAssigns([]func() error{
		func() error { return nil },
		func() error { return errBoom },
	})
	if err == nil || !errors.Is(err, errBoom) {
		t.Fatalf("error = %v, want boom", err)
	}
}

func TestDecodeGoListOutputBadJSON(t *testing.T) {
	t.Parallel()

	_, err := decodeGoListOutput(strings.NewReader("{"))
	if err == nil || !strings.Contains(err.Error(), "decode go list stream") {
		t.Fatalf("error = %v, want decode wrapper", err)
	}
}

func TestDecodeGoListItemBadField(t *testing.T) {
	t.Parallel()

	decoder := json.NewDecoder(strings.NewReader(`{"ImportPath":123}`))
	_, _, err := decodeGoListItem(decoder)

	if err == nil || !strings.Contains(err.Error(), "decode go list item") {
		t.Fatalf("error = %v, want item decode wrapper", err)
	}
}

func TestGoListWaitFailureDecodeBranch(t *testing.T) {
	t.Parallel()

	stderr := NewCappedBuffer(32)
	err := goListWaitFailure(t.Context(), &goListRun{stderr: &stderr}, &waitErrs{
		decode: errDecodeBoom,
		wait:   nil,
	})
	if err == nil || !errors.Is(err, errDecodeBoom) {
		t.Fatalf("error = %v, want decode failure", err)
	}
}

func TestRunGoListStdoutPipeFailure(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(t.Context(), goCommand, goListCommandName, goListJSONFlag, ".")
	cmd.Stdout = io.Discard

	stderr := NewCappedBuffer(32)
	_, err := startGoListCommand(cmd, &stderr)

	if err == nil || !strings.Contains(err.Error(), "prepare go list") {
		t.Fatalf("error = %v, want stdout pipe wrapper", err)
	}
}

func TestCreateTempProfileTMPDIRFailure(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(blocked, "missing"))

	_, err := createTempProfileContext(t.Context(), closeFile)
	if err == nil || !strings.Contains(err.Error(), "create temporary coverage profile") {
		t.Fatalf("error = %v, want create temp wrapper", err)
	}
}

func TestCreateHTMLProfileFileTMPDIRFailure(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(blocked, "missing"))

	_, _, err := createHTMLProfileFileContext(t.Context())
	if err == nil || !strings.Contains(err.Error(), "create temporary HTML coverage input") {
		t.Fatalf("error = %v, want HTML create wrapper", err)
	}
}

func TestCollectCreateTempProfileFailure(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(blocked, "missing"))

	_, err := New().Collect(t.Context(), &ports.CoverageRequest{Patterns: []string{"."}, TestArgs: nil})
	if err == nil || !strings.Contains(err.Error(), "collect coverage") {
		t.Fatalf("error = %v, want collect wrapper", err)
	}
}

func TestOpenHTMLReportCreateFailure(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(blocked, "missing"))

	err := openHTMLReport(t.Context(), &collectionrequest.HTMLOpenRequest{
		Profile: []byte("mode: atomic\n"),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want open HTML wrapper", err)
	}
}

func TestWriteAndCloseProfileWriteFailure(t *testing.T) {
	t.Parallel()

	err := writeAndCloseProfile(createClosedTempFile(t), []byte("mode: atomic\n"))
	if err == nil || !strings.Contains(err.Error(), "write temporary HTML coverage input") {
		t.Fatalf("error = %v, want write wrapper", err)
	}
}

func TestWriteAndCloseProfileCloseFailure(t *testing.T) {
	file := createTempFile(t)
	err := writeAndCloseProfileWith(file, []byte("mode: atomic\n"), func(closed *os.File) error {
		_ = closeFile(closed)

		return errCloseBoom
	})
	if err == nil || !errors.Is(err, errCloseBoom) {
		t.Fatalf("error = %v, want close boom", err)
	}
}

func TestFinishHTMLReportWriteFailure(t *testing.T) {
	t.Parallel()

	err := finishHTMLReport(
		t.Context(),
		createClosedTempFile(t),
		&collectionrequest.HTMLOpenRequest{
			Profile: []byte("mode: atomic\n"),
			Stdout:  io.Discard,
			Stderr:  io.Discard,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want finish HTML wrapper", err)
	}
}

func TestReadParsedCoverageErrorArms(t *testing.T) {
	t.Parallel()

	_, err := readParsedCoverageContext(
		t.Context(),
		filepath.Join(t.TempDir(), "missing.coverprofile"),
	)
	if err == nil || !strings.Contains(err.Error(), "read temporary coverage profile") {
		t.Fatalf("error = %v, want read wrapper", err)
	}

	path := filepath.Join(t.TempDir(), "bad.coverprofile")
	if writeErr := os.WriteFile(path, []byte("not-a-profile\n"), 0o600); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}

	_, err = readParsedCoverageContext(t.Context(), path)
	if err == nil || !strings.Contains(err.Error(), "parse coverage profile") {
		t.Fatalf("error = %v, want parse wrapper", err)
	}
}

func TestCollectFromProfileParseFailure(t *testing.T) {
	t.Parallel()

	dir := writeInternalGoModule(t)
	_, err := collectFromProfile(t.Context(), os.DevNull, &ports.CoverageRequest{
		Patterns: []string{dir},
		TestArgs: []string{"-run", "TestAdd"},
	})
	if err == nil || !strings.Contains(err.Error(), "collect from profile") {
		t.Fatalf("error = %v, want collect-from-profile wrapper", err)
	}
}

func TestOpenSucceedsWithValidProfile(t *testing.T) {
	t.Setenv("BROWSER", "true")

	dir := writeInternalGoModule(t)
	coverage, err := New().Collect(t.Context(), &ports.CoverageRequest{
		Patterns: []string{dir},
		TestArgs: []string{"-run", "TestAdd"},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	err = New().Open(t.Context(), &collectionrequest.HTMLOpenRequest{
		Profile: coverage.Profile,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
}

func TestFinishProfileScanReportsScannerError(t *testing.T) {
	t.Parallel()

	_, err := ParseProfile(&partialErrReader{
		data: []byte("mode: atomic\n"),
		err:  errBoom,
	})
	if err == nil || !strings.Contains(err.Error(), "read coverage profile") {
		t.Fatalf("error = %v, want scanner read wrapper", err)
	}
}

func TestCreateTempProfileCloseFailure(t *testing.T) {
	_, err := createTempProfileContext(t.Context(), func(file *os.File) error {
		_ = closeFile(file)

		return errCloseBoom
	})
	if err == nil || !strings.Contains(err.Error(), "create temporary coverage profile") {
		t.Fatalf("error = %v, want create/close wrapper", err)
	}
}

func TestCloseCreatedTempProfileSuccess(t *testing.T) {
	path, err := closeCreatedTempProfileWithContext(t.Context(), createTempFile(t), closeFile)
	if err != nil {
		t.Fatalf("closeCreatedTempProfile: %v", err)
	}

	if path == emptyString {
		t.Fatal("closeCreatedTempProfile returned empty path")
	}
}

func TestCreateTempProfileCreateFailure(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing", "nested"))

	_, err := createTempProfileContext(t.Context(), closeFile)
	if err == nil || !strings.Contains(err.Error(), "create temporary coverage profile") {
		t.Fatalf("error = %v, want create failure", err)
	}
}

func TestCreateTempProfileWithCreateFailure(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing", "nested"))

	_, err := createTempProfileContext(t.Context(), closeFile)
	if err == nil || !strings.Contains(err.Error(), "create temporary coverage profile") {
		t.Fatalf("error = %v, want create failure", err)
	}
}

func failedGoTestRun(profilePath, output string) *goTestRun {
	buf := NewCappedBuffer(commandOutputLimit)
	_, _ = buf.Write([]byte(output))

	return &goTestRun{
		request:     nil,
		output:      &buf,
		profilePath: profilePath,
	}
}

func createTempFile(t *testing.T) *os.File {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "gotool-*.tmp")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}

	t.Cleanup(func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	})

	return file
}

func createClosedTempFile(t *testing.T) *os.File {
	t.Helper()

	file := createTempFile(t)
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	return file
}

func writeInternalGoModule(t *testing.T) string {
	t.Helper()

	temp := t.TempDir()
	suffix := filepath.Base(filepath.Dir(temp)) + "-" + filepath.Base(temp)
	dir := filepath.Join(
		".",
		"coverlint-fixture-"+strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+"-"+suffix,
	)

	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("RemoveAll %s: %v", dir, err)
		}
	})

	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(
		"package fixture\n\nfunc Add(a, b int) int { return a + b }\n",
	), 0o600); err != nil {
		t.Fatalf("WriteFile calc.go: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(`package fixture

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("bad add")
	}
}
`), 0o600); err != nil {
		t.Fatalf("WriteFile calc_test.go: %v", err)
	}

	return "./" + filepath.Base(dir)
}

func TestRetryProcessHelper(t *testing.T) {
	if os.Getenv("COVERLINT_PROCESS_HELPER") != "1" {
		t.Parallel()
		return
	}
	os.Exit(7)
}

func failingCommand(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRetryProcessHelper$")
	cmd.Env = append(os.Environ(), "COVERLINT_PROCESS_HELPER=1")
	return cmd
}

func TestStartUsesFreshCommandsAndWaitsOnce(t *testing.T) {
	t.Parallel()
	var commands []*exec.Cmd
	starts := 0
	cmd, err := startCommand(t.Context(), func() *exec.Cmd {
		cmd := failingCommand(t.Context())
		commands = append(commands, cmd)
		return cmd
	}, func(cmd *exec.Cmd) error {
		starts++
		if starts < 3 {
			return &os.PathError{Op: "fork", Path: cmd.Path, Err: syscall.EAGAIN}
		}
		return cmd.Start()
	})
	if err != nil {
		t.Fatal(err)
	}
	if starts != 3 || commands[0] == commands[1] || commands[1] == commands[2] {
		t.Fatalf("commands were reused: %v, starts %d", commands, starts)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("expected process exit error")
	}
}

func TestRunCommandDoesNotRetryExitFailure(t *testing.T) {
	t.Parallel()
	calls := 0
	err := runCommand(t.Context(), func() *exec.Cmd { calls++; return failingCommand(t.Context()) })
	var exitErr *exec.ExitError
	if calls != 1 || !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("run error = %v, calls %d", err, calls)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = runCommand(ctx, func() *exec.Cmd { calls++; return failingCommand(ctx) })
	if calls != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v, calls %d", err, calls)
	}
}

func TestToolOperationsNeverRetryStartedProcesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "calls")
	script := `#!/bin/sh
printf '%s\n' "$1" >> "$COVERLINT_CALLS"
case "$1" in
test)
  printf 'mode: atomic\nexample.com/fixture/calc.go:1.1,1.2 1 0\n' > "$5"
  if [ "$COVERLINT_BAD_PROFILE" = 1 ]; then printf 'bad profile\n' > "$5"; fi
  printf '%s\n' '{"Action":"fail","Package":"example.com/fixture","Test":"TestFail"}'
  exit 1
  ;;
list)
  printf '{bad json'
  exit 0
  ;;
tool)
  exit 1
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("COVERLINT_CALLS", marker)
	tool := New()
	coverage, err := tool.Collect(
		t.Context(),
		&ports.CoverageRequest{Patterns: []string{"./..."}},
	)
	if err != nil || !coverage.TestsFailed || len(coverage.Failures) == 0 {
		t.Fatalf("failed tests should remain a soft failure: %#v, %v", coverage, err)
	}
	t.Setenv("COVERLINT_BAD_PROFILE", "1")
	if _, err := tool.Collect(
		t.Context(),
		&ports.CoverageRequest{Patterns: []string{"./..."}},
	); err == nil {
		t.Fatal("expected invalid coverage profile to remain a hard failure")
	}
	if _, err := tool.List(
		t.Context(),
		&ports.PackageRequest{Patterns: []string{"./..."}},
	); err == nil {
		t.Fatal("expected JSON decode error")
	}
	err = tool.Open(
		t.Context(),
		&collectionrequest.HTMLOpenRequest{
			Profile: coverage.Profile,
			Stdout:  io.Discard,
			Stderr:  io.Discard,
		},
	)
	if err == nil {
		t.Fatal("expected HTML command failure")
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "test\ntest\nlist\ntool\n" {
		t.Fatalf("executions = %q, %v", data, err)
	}
}

func TestStartPermanentFailure(t *testing.T) {
	t.Parallel()
	calls := 0
	cmd, err := startCommand(t.Context(), func() *exec.Cmd { return failingCommand(t.Context()) },
		func(*exec.Cmd) error { calls++; return os.ErrPermission })
	if cmd != nil || calls != 1 || !errors.Is(err, os.ErrPermission) ||
		!strings.Contains(err.Error(), "start process") {
		t.Fatalf("start = %v, %v, calls %d", cmd, err, calls)
	}
}

func TestListMissingExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	packages, err := New().List(t.Context(), &ports.PackageRequest{Patterns: []string{"."}})
	if packages != nil || !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("missing executable = %v, %v", packages, err)
	}
}

func TestReadProfileCloseFailureIsPermanent(t *testing.T) {
	t.Parallel()
	file := createClosedTempFile(t)
	calls := 0
	err := retry.Run(
		t.Context(),
		func() error { calls++; _, err := readAndCloseProfile(file); return err },
	)
	if calls != 1 || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("close failure = %v, calls %d", err, calls)
	}
	ctx, cancel := context.WithCancelCause(t.Context())
	cause := errors.New("coverage caller stopped")
	cancel(cause)
	if err := goCoverError(ctx, os.ErrPermission); !errors.Is(err, cause) {
		t.Fatalf("cover cause = %v", err)
	}
}

func TestTemporaryCleanupAfterCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture requires a POSIX shell")
	}
	dir := t.TempDir()
	temp := t.TempDir()
	marker := filepath.Join(dir, "started")
	script := "#!/bin/sh\nprintf started > \"$COVERLINT_STARTED\"\nwhile :; do :; done\n"
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TMPDIR", temp)
	t.Setenv("COVERLINT_STARTED", marker)
	for _, open := range []bool{false, true} {
		ctx, stop := context.WithTimeout(t.Context(), 30*time.Second)
		t.Cleanup(stop)
		ctx, cancel := context.WithCancelCause(ctx)
		cause := errors.New("cancel after process start")
		done := make(chan error, 1)
		go func() {
			if open {
				done <- New().Open(ctx, &collectionrequest.HTMLOpenRequest{Profile: []byte("mode: atomic\n"), Stdout: io.Discard, Stderr: io.Discard})
			} else {
				_, err := New().Collect(ctx, &ports.CoverageRequest{Patterns: []string{"."}})
				done <- err
			}
		}()
		waitForProcessMarker(t, ctx, marker)
		cancel(cause)
		err := <-done
		stop()
		if !errors.Is(err, cause) {
			t.Fatalf("operation canceled = %v", err)
		}
		entries, err := os.ReadDir(temp)
		if err != nil || len(entries) != 0 {
			t.Fatalf("temporary files leaked: %v, %v", entries, err)
		}
		if err := os.Remove(marker); err != nil {
			t.Fatal(err)
		}
	}
}

func waitForProcessMarker(t *testing.T, ctx context.Context, marker string) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("process never signaled startup: %v", context.Cause(ctx))
		case <-ticker.C:
			_, err := os.Stat(marker)
			if err == nil {
				return
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
	}
}

func TestCaptureAttributesParallelFailures(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	events := []testEvent{
		{
			Action:  "output",
			Package: "a",
			Test:    "TestCalc/case",
			Output:  "input: 2\ngot: 3\nwant: 4\n",
		},
		{Action: "output", Package: "b", Test: "TestPass", Output: "passing log\n"},
		{Action: "fail", Package: "a", Test: "TestCalc/case"},
		{Action: "fail", Package: "b", Test: "TestSilent"},
		{Action: "fail", Package: "a"},
	}
	writeCaptureEvents(t, capture, events)
	failures := captureFailures(capture)
	if len(failures) != 3 || failures[0].Test != "TestCalc/case" ||
		failures[1].Test != "TestSilent" {
		t.Fatalf("failures = %#v", failures)
	}
	if failures[0].Output != events[0].Output || failures[1].Output != "" {
		t.Fatalf("failure output = %#v", failures)
	}
	if !strings.Contains(buffer.String(), "passing log") ||
		strings.Contains(buffer.String(), `"Action"`) {
		t.Fatalf("text report = %q", buffer.String())
	}
}

func TestCaptureGroupsInterleavedPackageOutput(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	writeCaptureEvents(t, capture, []testEvent{
		{Action: "output", Package: "a", Output: "=== RUN   TestA\n"},
		{Action: "output", Package: "b", Output: "=== RUN   TestB\n"},
		{Action: "output", Package: "b", Output: "--- PASS: TestB (0.00s)\nok  b\n"},
		{Action: "output", Package: "a", Output: "--- PASS: TestA (0.00s)\nok  a\n"},
	})
	captureGroupOutput(capture)
	want := "=== RUN   TestA\n--- PASS: TestA (0.00s)\nok  a\n" +
		"=== RUN   TestB\n--- PASS: TestB (0.00s)\nok  b\n"
	if buffer.String() != want {
		t.Fatalf("report = %q, want %q", buffer.String(), want)
	}
}

func TestCaptureRetainsNamesAfterTruncation(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(5)
	capture := newTestCapture(&buffer)
	writeCaptureEvents(t, capture, []testEvent{
		{Action: "output", Package: "a", Test: "TestFirst", Output: "long output\n"},
		{Action: "fail", Package: "a", Test: "TestFirst"},
		{Action: "fail", Package: "b", Test: "TestLater"},
		{Action: "fail", Package: "b"},
	})
	if failures := captureFailures(capture); len(failures) != 4 || failures[1].Test != "TestLater" {
		t.Fatalf("failures = %#v", failures)
	}
	if packages := captureFailedPackages(capture, nil); len(packages) != 2 || packages[1] != "b" {
		t.Fatalf("packages = %#v", packages)
	}
	captureGroupOutput(capture)
	if !strings.Contains(buffer.String(), "output truncated") {
		t.Fatalf("output = %q", buffer.String())
	}
}

func TestCaptureHandlesRawAndOversizedLines(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	_, err := testCaptureOutput(capture).Write(
		[]byte("compiler error\n" + strings.Repeat("x", maxScannerBufferSize+2) + "\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	writeCaptureEvents(t, capture, []testEvent{{Action: "fail", Package: "a", Test: "TestAfter"}})
	captureGroupOutput(capture)
	if !strings.Contains(buffer.String(), "compiler error") ||
		captureFailures(capture)[0].Test != "TestAfter" {
		t.Fatalf("capture = %#v, output = %q", captureFailures(capture), buffer.String())
	}
}

func TestCapturePackageFailureOmitsPassingLogs(t *testing.T) {
	t.Parallel()

	buffer := NewCappedBuffer(commandOutputLimit)
	capture := newTestCapture(&buffer)
	writeCaptureEvents(t, capture, []testEvent{
		{Action: "output", Package: "a", Test: "TestPass", Output: "passing log\n"},
		{Action: "pass", Package: "a", Test: "TestPass"},
		{Action: "output", Package: "a", Test: "TestPanic", Output: "panic: broken\n"},
		{Action: "output", Package: "a", Output: "FAIL\n"},
		{Action: "fail", Package: "a"},
	})
	output := captureFailures(capture)[0].Output
	if strings.Contains(output, "passing log") || !strings.Contains(output, "panic: broken") {
		t.Fatalf("package failure = %q", output)
	}
}

func writeCaptureEvents(t *testing.T, capture *testCaptureState, events []testEvent) {
	t.Helper()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		// Split the line to exercise arbitrary command-pipe write boundaries.
		for _, chunk := range [][]byte{data[:len(data)/2], append(data[len(data)/2:], '\n')} {
			if _, err := testCaptureOutput(capture).Write(chunk); err != nil {
				t.Fatal(err)
			}
		}
	}
	captureFinish(capture)
}

func TestParseTestReport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, input, pkg, test, detail string
		failed                         bool
	}{
		{name: "passing", input: "ok\texample.com/pkg\t0.1s\n"},
		{
			name:  "ordinary log",
			input: "failure expected\n    FAIL is only a word\nok\texample.com/pkg\t0.1s\n",
		},
		{name: "empty"},
		{name: "blank", input: "\n\n"},
		{
			name:   "package",
			input:  "panic: broken\nFAIL\texample.com/pkg\t0.1s\n",
			pkg:    "example.com/pkg",
			detail: "panic: broken",
			failed: true,
		},
		{name: "standalone", input: "FAIL\n", detail: "FAIL", failed: true},
		{
			name:   "unattributed",
			input:  "--- FAIL: TestCalc (0.00s)\n    calc_test.go:9: got=1 want=2\n",
			test:   "TestCalc",
			detail: "got=1 want=2",
			failed: true,
		},
		{
			name:   "subtest",
			input:  "--- FAIL: TestCalc/case (0.00s)\n    calc_test.go:9: got=1 want=2\nFAIL\nFAIL\texample.com/pkg\t0.1s\n",
			pkg:    "example.com/pkg",
			test:   "TestCalc/case",
			detail: "got=1 want=2",
			failed: true,
		},
		{
			name:   "JSON",
			input:  "{\"Action\":\"output\",\"Package\":\"example.com/pkg\",\"Test\":\"TestCalc\",\"Output\":\"got=1 want=2\\n\"}\n{\"Action\":\"fail\",\"Package\":\"example.com/pkg\",\"Test\":\"TestCalc\"}",
			pkg:    "example.com/pkg",
			test:   "TestCalc",
			detail: "got=1 want=2",
			failed: true,
		},
		{
			name:  "passing JSON",
			input: "{\"Action\":\"output\",\"Output\":\"FAIL is expected\\n\"}\n{\"Action\":\"pass\",\"Package\":\"example.com/pkg\"}\n",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			report, err := parseTestReport(
				bufio.NewScanner(strings.NewReader(testCase.input)),
				"report.txt",
			)
			if err != nil || report.TestsFailed != testCase.failed {
				t.Fatalf("report=%#v err=%v", report, err)
			}
			if !testCase.failed {
				return
			}
			if len(report.Failures) != 1 {
				t.Fatalf("failures=%#v", report.Failures)
			}
			failure := report.Failures[0]
			if failure.ImportPath != testCase.pkg || failure.Test != testCase.test ||
				!strings.Contains(failure.Output, testCase.detail) ||
				!strings.Contains(failure.Output, "report.txt") {
				t.Fatalf("failure=%#v", failure)
			}
		})
	}
}

func TestReadTestReportOptionalAndErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, path := range []string{"", filepath.Join(dir, "missing.txt")} {
		report, err := ReadTestReport(path)
		if err != nil || report.TestsFailed {
			t.Fatalf("path=%q report=%#v err=%v", path, report, err)
		}
	}
	if _, err := ReadTestReport(dir); err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("directory error=%v", err)
	}
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte("FAIL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ReadTestReport(path)
	if err != nil || !report.TestsFailed {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if _, err := ReadTestReport(
		filepath.Join(path, "child"),
	); err == nil ||
		!strings.Contains(err.Error(), "open test report") {
		t.Fatalf("invalid parent error=%v", err)
	}
}

func TestTextReportPackageGrouping(t *testing.T) {
	t.Parallel()
	input := "--- FAIL: TestFirst (0s)\n    first assertion\nFAIL\nFAIL\ta/pkg\t0.1s\n--- PASS: TestGood (0s)\nok\tb/pkg\t0.1s\n--- FAIL: TestLast (0s)\n    last assertion\nFAIL\tc/pkg\t0.1s\nFAIL\n"
	report, err := parseTestReport(bufio.NewScanner(strings.NewReader(input)), "grouped.txt")
	if err != nil || len(report.Failures) != 2 || len(report.FailedPackages) != 2 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if report.Failures[0].ImportPath != "a/pkg" || report.Failures[1].ImportPath != "c/pkg" {
		t.Fatalf("failures=%#v", report.Failures)
	}
}

func TestTextReportTruncationKeepsNames(t *testing.T) {
	t.Parallel()
	input := strings.Repeat(
		"ordinary output\n",
		commandOutputLimit/10,
	) + "--- FAIL: TestLate (0s)\n    assertion\nFAIL\tpkg\t0.1s\n"
	report, err := parseTestReport(bufio.NewScanner(strings.NewReader(input)), "large.txt")
	if err != nil || len(report.Failures) != 1 || report.Failures[0].Test != "TestLate" ||
		!strings.Contains(report.Failures[0].Output, "truncated") {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if name := textFailureName(textTestFailPrefix); name != "" {
		t.Fatalf("empty name=%q", name)
	}
}
