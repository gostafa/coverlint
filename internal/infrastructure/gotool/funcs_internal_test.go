// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
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

	_, err := readTempProfile(filepath.Join(t.TempDir(), "missing.coverprofile"))

	if err == nil || !strings.Contains(err.Error(), "open temporary coverage profile") {
		t.Fatalf("error = %v, want open wrapper", err)
	}
}

func TestReadTempProfileWrapsReadError(t *testing.T) {
	t.Parallel()

	_, err := readTempProfile(t.TempDir())

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

	buffer := CappedBuffer{
		buffer: &failBuffer{err: errWriteBoom},
		limit:  8,
	}

	_, err := buffer.Write([]byte("payload"))
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
	err := closeTempProfileError(missing, errBoom)

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
	_, err := closeCreatedTempProfile(file)

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

	err := removeFile(filepath.Join(t.TempDir(), "missing"))
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

	cmd := exec.CommandContext(t.Context(), goCommand, "list", "-json", ".")
	cmd.Stdout = io.Discard

	stderr := NewCappedBuffer(32)
	_, err := runGoList(t.Context(), cmd, &stderr)

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

	_, err := createTempProfile()
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

	_, _, err := createHTMLProfileFile()
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

	_, err := New().Collect(t.Context(), &outbound.CoverageRequest{Patterns: []string{"."}, TestArgs: nil})
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

	err := openHTMLReport(t.Context(), &outbound.HTMLOpenRequest{
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
	original := closeFile
	t.Cleanup(func() { closeFile = original })

	closeFile = func(file *os.File) error {
		_ = original(file)

		return errCloseBoom
	}

	file := createTempFile(t)
	err := writeAndCloseProfile(file, []byte("mode: atomic\n"))
	if err == nil || !errors.Is(err, errCloseBoom) {
		t.Fatalf("error = %v, want close boom", err)
	}
}

func TestFinishHTMLReportWriteFailure(t *testing.T) {
	t.Parallel()

	err := finishHTMLReport(t.Context(), createClosedTempFile(t), &outbound.HTMLOpenRequest{
		Profile: []byte("mode: atomic\n"),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "open HTML coverage report") {
		t.Fatalf("error = %v, want finish HTML wrapper", err)
	}
}

func TestReadParsedCoverageErrorArms(t *testing.T) {
	t.Parallel()

	_, err := readParsedCoverage(filepath.Join(t.TempDir(), "missing.coverprofile"))
	if err == nil || !strings.Contains(err.Error(), "read temporary coverage profile") {
		t.Fatalf("error = %v, want read wrapper", err)
	}

	path := filepath.Join(t.TempDir(), "bad.coverprofile")
	if writeErr := os.WriteFile(path, []byte("not-a-profile\n"), 0o600); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}

	_, err = readParsedCoverage(path)
	if err == nil || !strings.Contains(err.Error(), "parse coverage profile") {
		t.Fatalf("error = %v, want parse wrapper", err)
	}
}

func TestCollectFromProfileParseFailure(t *testing.T) {
	t.Parallel()

	dir := writeInternalGoModule(t)
	_, err := collectFromProfile(t.Context(), os.DevNull, &outbound.CoverageRequest{
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
	coverage, err := New().Collect(t.Context(), &outbound.CoverageRequest{
		Patterns: []string{dir},
		TestArgs: []string{"-run", "TestAdd"},
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	err = New().Open(t.Context(), &outbound.HTMLOpenRequest{
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
	original := closeFile
	t.Cleanup(func() { closeFile = original })

	closeFile = func(file *os.File) error {
		_ = original(file)

		return errCloseBoom
	}

	_, err := createTempProfile()
	if err == nil || !strings.Contains(err.Error(), "create temporary coverage profile") {
		t.Fatalf("error = %v, want create/close wrapper", err)
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
	dir := filepath.Join(".", "coverlint-fixture-"+strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+"-"+suffix)

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
