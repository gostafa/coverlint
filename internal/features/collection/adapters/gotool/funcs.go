// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gostafa/coverlint/internal/features/collection/domain"
	"github.com/gostafa/coverlint/internal/features/collection/ports"
	collectionrequest "github.com/gostafa/coverlint/internal/features/collection/ports/request"
	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

// ListArgsForTestArgs returns go list-compatible build flags from go test flags.
func ListArgsForTestArgs(testArgs []string) []string {
	listArgs := make([]string, zero, len(testArgs))

	for index := zero; index < len(testArgs); index++ {
		args, consumed := listArgsForTestArg(testArgs, index)

		index += consumed

		listArgs = append(listArgs, args...)
	}

	return listArgs
}

// New creates a Go toolchain coverage adapter.
func New() *Adapter {
	adapter := Adapter(nil)

	return &adapter
}

// NewCappedBuffer creates a capped output buffer.
func NewCappedBuffer(limit int) CappedBuffer {
	return CappedBuffer{
		buffer:    &bytes.Buffer{},
		limit:     limit,
		truncated: false,
	}
}

// ParseProfile reads Go coverage profile blocks.
func ParseProfile(reader io.Reader) ([]domain.Block, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, initialScannerBufferSize), maxScannerBufferSize)

	err := readProfileHeader(scanner)
	if err != nil {
		return nil, fmt.Errorf(errParseCoverageProfileFormat, err)
	}

	blocks, err := finishProfileScan(scanner)
	if err != nil {
		return nil, fmt.Errorf(errParseCoverageProfileFormat, err)
	}

	return blocks, nil
}

// ReadTestReport reads optional existing Go text or JSON test results.
func ReadTestReport(path string) (domain.Coverage, error) {
	coverage, err := ReadTestReportContext(context.Background(), path)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("read test report: %w", err)
	}

	return coverage, nil
}

// ReadTestReportContext reads an optional report with cancellable retry waits.
func ReadTestReportContext(ctx context.Context, path string) (domain.Coverage, error) {
	var coverage domain.Coverage

	if path == emptyString {
		return coverage, nil
	}

	coverage, err := readTestReportFile(ctx, path)

	if errors.Is(err, os.ErrNotExist) && context.Cause(ctx) == nil {
		return coverage, nil
	}

	if err != nil {
		return domain.Coverage{}, fmt.Errorf("open test report %q: %w", path, err)
	}

	return coverage, nil
}

func absolutePackageFile(dir, filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}

	return filepath.Join(dir, filename)
}

func annotateCapturedFailures(coverage *domain.Coverage, capture *testCaptureState) {
	if capture == nil {
		return
	}

	coverage.Failures = captureFailures(capture)
	coverage.FailedPackages = captureFailedPackages(capture, coverage.FailedPackages)
}

func annotateFailedCoverage(coverage *domain.Coverage, output string) domain.Coverage {
	coverage.FailedPackages = parseFailedPackages(output)
	coverage.TestOutput = strings.TrimSpace(output)
	coverage.TestsFailed = true

	return *coverage
}

func appendParsedFailPackage(packages []string, seen map[string]struct{}, line string) []string {
	pkg, ok := parseFailPackageLine(line)

	if !ok {
		return packages
	}

	if _, exists := seen[pkg]; exists {
		return packages
	}

	seen[pkg] = struct{}{}

	return append(packages, pkg)
}

func appendTextFailureOutput(text *textTestReport, line string) {
	if len(text.failures) == zero || text.output.truncated {
		return
	}

	if !indentedReportLine(line) {
		return
	}

	text.failures[len(text.failures)-one].Output += line + "\n"
}

func appendTextReportFailures(text *textTestReport, result *domain.Coverage, pkg string) {
	result.TestsFailed = true
	result.FailedPackages = appendUniquePackage(result.FailedPackages, pkg)

	if len(text.failures) == zero {
		text.failures = append(text.failures, domain.TestFailure{
			Output: text.output.String(), ImportPath: emptyString, Test: emptyString,
		})
	}

	for i := range text.failures {
		text.failures[i].ImportPath = pkg

		if text.output.truncated && text.failures[i].Test != emptyString {
			text.failures[i].Output += truncationSuffix
		}

		result.Failures = append(result.Failures, text.failures[i])
	}
}

func appendTextReportOutput(text *textTestReport, line string) {
	if text.output == nil {
		output := NewCappedBuffer(commandOutputLimit)

		text.output = &output
	}

	writeCappedBytes(&cappedWrite{
		buffer: text.output.buffer, truncated: &text.output.truncated, limit: text.output.limit,
	}, []byte(line+"\n"))
}

func appendUniquePackage(packages []string, pkg string) []string {
	if pkg == emptyString || slices.Contains(packages, pkg) {
		return packages
	}

	return append(packages, pkg)
}

func assignGoListPackage(raw map[string]json.RawMessage) (goListPackage, error) {
	var pkg goListPackage

	assigns := []func() error{
		func() error { return unmarshalGoListString(raw, jsonKeyImportPath, &pkg.ImportPath) },
		func() error { return unmarshalGoListString(raw, jsonKeyDir, &pkg.Dir) },
		func() error { return unmarshalGoListStrings(raw, jsonKeyGoFiles, &pkg.GoFiles) },
		func() error { return unmarshalGoListStrings(raw, jsonKeyCgoFiles, &pkg.CgoFiles) },
		func() error { return unmarshalGoListStrings(raw, jsonKeyTestGoFiles, &pkg.TestGoFiles) },
	}

	err := runGoListAssigns(assigns)
	if err != nil {
		return goListPackage{}, fmt.Errorf(errUnmarshalGoListFormat, err)
	}

	return pkg, nil
}

func buildProfileBlock(file string, fields []string, lineNumber int) (domain.Block, error) {
	statements, err := parseStatementCount(fields[one])
	if err != nil {
		return domain.Block{}, fmt.Errorf(errWrapLine, errInvalidStatementCount, lineNumber)
	}

	covered, err := parseCovered(fields[2])
	if err != nil {
		return domain.Block{}, fmt.Errorf(errWrapLine, errInvalidExecutionCount, lineNumber)
	}

	return domain.Block{
		File:       file,
		Position:   fields[zero],
		Statements: statements,
		Covered:    covered,
	}, nil
}

func cappedSuffixes() [two]string {
	return [two]string{emptyString, truncationSuffix}
}

func captureAppendFallbacks(
	capture *testCaptureState,
	result []domain.TestFailure,
) []domain.TestFailure {
	if len(capture.packageLogs[emptyString]) > zero {
		result = append(result, captureFallback(string(capture.packageLogs[emptyString])))
	}

	if capture.output.truncated {
		result = append(result, captureFallback(truncationSuffix))
	}

	return result
}

func captureConsume(capture *testCaptureState, value byte) {
	if value == '\n' {
		captureFinish(capture)

		return
	}

	if capture.drop {
		return
	}

	if len(capture.line) >= maxScannerBufferSize {
		capture.drop = true
		capture.output.truncated = true

		return
	}

	capture.line = append(capture.line, value)
}

func captureFailedPackages(capture *testCaptureState, fallback []string) []string {
	result := make([]string, zero, len(capture.failed)+len(fallback))

	for i := range capture.failed {
		result = appendUniquePackage(result, capture.failed[i].pkg)
	}

	for i := range fallback {
		result = appendUniquePackage(result, fallback[i])
	}

	return result
}

func captureFailure(capture *testCaptureState, key *testKey) domain.TestFailure {
	output := capture.logs[*key]

	if key.test == emptyString {
		output = capturePackageOutput(capture, key.pkg)
	}

	return domain.TestFailure{ImportPath: key.pkg, Test: key.test, Output: string(output)}
}

func captureFailures(capture *testCaptureState) []domain.TestFailure {
	result := make([]domain.TestFailure, zero, len(capture.failed))

	for i := range capture.failed {
		result = append(result, captureFailure(capture, &capture.failed[i]))
	}

	return captureAppendFallbacks(capture, result)
}

func captureFallback(output string) domain.TestFailure {
	return domain.TestFailure{ImportPath: emptyString, Test: emptyString, Output: output}
}

func captureFinish(capture *testCaptureState) {
	if !capture.drop && len(capture.line) > zero {
		captureRecord(capture, capture.line)
	}

	capture.line = capture.line[:zero]
	capture.drop = false
}

// captureGroupOutput preserves event order within each package for text report parsers.
func captureGroupOutput(capture *testCaptureState) {
	output := make([]byte, zero, capture.output.buffer.Len())

	for i := range capture.reportOrder {
		output = append(output, capture.reportLogs[capture.reportOrder[i]]...)
	}

	capture.output.buffer = bytes.NewBuffer(output)
}

func capturePackageOutput(capture *testCaptureState, pkg string) []byte {
	output := slices.Clone(capture.packageLogs[pkg])

	for i := range capture.order {
		key := &capture.order[i]

		if capturePendingTest(capture, key, pkg) {
			output = append(output, capture.logs[*key]...)
		}
	}

	return output
}

func capturePendingTest(capture *testCaptureState, key *testKey, pkg string) bool {
	return key.pkg == pkg && key.test != emptyString && !capture.passed[*key]
}

func captureRecord(capture *testCaptureState, line []byte) {
	var event testEvent

	if json.Unmarshal(line, &event) != nil || event.Action == emptyString {
		key := testKey{pkg: emptyString, test: emptyString}

		captureRecordOutput(capture, &key, string(line)+"\n")

		return
	}

	captureRecordEvent(capture, &event)
}

func captureRecordEvent(capture *testCaptureState, event *testEvent) {
	key := testKey{pkg: event.Package, test: event.Test}

	if event.Action == testActionPass {
		capture.passed[key] = true
	}

	if event.Action == testActionFail {
		capture.failed = append(capture.failed, key)
	}

	if event.Output != emptyString {
		captureRecordOutput(capture, &key, event.Output)
	}
}

func captureRecordOutput(capture *testCaptureState, key *testKey, output string) {
	before := capture.output.buffer.Len()
	writeCappedBytes(
		&cappedWrite{
			buffer:    capture.output.buffer,
			truncated: &capture.output.truncated,
			limit:     capture.output.limit,
		},
		[]byte(output),
	)

	stored := capture.output.buffer.Len() - before

	if stored > zero {
		captureRecordReportOutput(capture, key.pkg, output[:stored])
		captureRecordTestOutput(capture, key, output[:stored])
		captureRecordPackageOutput(capture, key, output[:stored])
	}
}

func captureRecordPackageOutput(capture *testCaptureState, key *testKey, output string) {
	if key.test == emptyString {
		capture.packageLogs[key.pkg] = append(capture.packageLogs[key.pkg], output...)
	}
}

func captureRecordReportOutput(capture *testCaptureState, pkg, output string) {
	if _, exists := capture.reportLogs[pkg]; !exists {
		capture.reportOrder = append(capture.reportOrder, pkg)
	}

	capture.reportLogs[pkg] = append(capture.reportLogs[pkg], output...)
}

func captureRecordTestOutput(capture *testCaptureState, key *testKey, output string) {
	if _, exists := capture.logs[*key]; !exists {
		capture.order = append(capture.order, *key)
	}

	capture.logs[*key] = append(capture.logs[*key], output...)
}

func closeCreatedTempProfileWithContext(
	ctx context.Context,
	profile *os.File,
	closer func(*os.File) error,
) (string, error) {
	profilePath := profile.Name()

	err := closer(profile)
	if err != nil {
		return emptyString, fmt.Errorf(
			errCloseCreatedTempProfile,
			closeTempProfileErrorContext(ctx, profilePath, err),
		)
	}

	return profilePath, nil
}

func closeFile(file *os.File) error {
	err := file.Close()
	if err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

func closeHTMLInput(file *os.File) error {
	err := closeHTMLInputWith(file, closeFile)
	if err != nil {
		return fmt.Errorf(errCloseHTMLInputFormat, err)
	}

	return nil
}

func closeHTMLInputErr(file *os.File) error {
	err := closeHTMLInputErrWith(file, closeFile)
	if err != nil {
		return fmt.Errorf(errCloseHTMLInputFormat, err)
	}

	return nil
}

func closeHTMLInputErrWith(file *os.File, closer func(*os.File) error) error {
	err := closer(file)
	if err == nil {
		return nil
	}

	return fmt.Errorf(errCloseHTMLInputFormat, err)
}

func closeHTMLInputWith(file *os.File, closer func(*os.File) error) error {
	err := closer(file)
	if err != nil {
		return fmt.Errorf(errCloseHTMLInputFormat, err)
	}

	return nil
}

func closeTempProfileErrorContext(ctx context.Context, profilePath string, err error) error {
	return errors.Join(
		fmt.Errorf(errCloseTempProfileFormat, err),
		removeFileContext(ctx, profilePath),
	)
}

func collectFailedGoTest(
	ctx context.Context,
	run *goTestRun,
	testErr error,
) (domain.Coverage, error) {
	output := run.output.String()
	hardErr := goTestError(ctx, testErr, output)

	if ctx.Err() != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, hardErr)
	}

	coverage, err := readParsedCoverageContext(ctx, run.profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, hardErr)
	}

	annotateFailedCoverage(&coverage, output)
	annotateCapturedFailures(&coverage, run.capture)

	return coverage, nil
}

func collectFromProfile(
	ctx context.Context,
	profilePath string,
	request *ports.CoverageRequest,
) (domain.Coverage, error) {
	output := NewCappedBuffer(commandOutputLimit)
	run := &goTestRun{profilePath: profilePath, request: request, output: &output, capture: nil}

	run.capture = newTestCapture(&output)

	testErr := runGoTest(ctx, run)
	captureFinish(run.capture)
	captureGroupOutput(run.capture)

	coverage, err := collectGoTestResult(ctx, run, testErr)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, err)
	}

	return coverage, nil
}

func collectGoTestResult(
	ctx context.Context,
	run *goTestRun,
	testErr error,
) (domain.Coverage, error) {
	var (
		coverage domain.Coverage
		err      error
	)

	if testErr != nil {
		coverage, err = collectFailedGoTest(ctx, run, testErr)
	} else {
		coverage, err = collectPassedGoTestContext(ctx, run)
	}

	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errWrap, err)
	}

	return coverage, nil
}

func collectPassedGoTestContext(ctx context.Context, run *goTestRun) (domain.Coverage, error) {
	coverage, err := readParsedCoverageContext(ctx, run.profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, err)
	}

	coverage.TestOutput = strings.TrimSpace(run.output.String())

	return coverage, nil
}

func createCoverageProfile(ctx context.Context) (string, error) {
	cause := context.Cause(ctx)
	if cause != nil {
		return emptyString, fmt.Errorf(
			errCollectCoverageFormat,
			goTestError(ctx, cause, emptyString),
		)
	}

	path, err := createTempProfileContext(ctx, closeFile)
	if err != nil {
		return emptyString, fmt.Errorf(errCreateTempProfileFormat, err)
	}

	return path, nil
}

func createHTMLProfileFileContext(ctx context.Context) (*os.File, string, error) {
	file, err := retry.CreateTemp(ctx, "coverlint-web-*.coverprofile")
	if err != nil {
		return nil, emptyString, fmt.Errorf("create temporary HTML coverage input: %w", err)
	}

	profilePath := file.Name()

	return file, profilePath, nil
}

func createTempProfileContext(ctx context.Context, closer func(*os.File) error) (string, error) {
	profile, err := retry.CreateTemp(ctx, tempProfilePattern)
	if err != nil {
		return emptyString, fmt.Errorf(errCreateTempProfileFormat, err)
	}

	profilePath, err := closeCreatedTempProfileWithContext(ctx, profile, closer)
	if err != nil {
		return emptyString, fmt.Errorf(errCreateTempProfileFormat, err)
	}

	return profilePath, nil
}

func decodeGoListItem(decoder *json.Decoder) (item goListPackage, done bool, err error) {
	var raw map[string]json.RawMessage

	raw, done, err = readGoListObject(decoder)
	if err != nil {
		return goListPackage{}, true, fmt.Errorf(errDecodeGoListItem, err)
	}

	if done {
		return item, true, nil
	}

	item, err = assignGoListPackage(raw)
	if err != nil {
		return goListPackage{}, false, fmt.Errorf(errDecodeGoListItem, err)
	}

	return item, false, nil
}

func decodeGoListOutput(stdout io.Reader) ([]domain.Package, error) {
	decoder := json.NewDecoder(stdout)
	packages := make([]domain.Package, zero, initialPackageCap)

	for {
		item, done, err := decodeGoListItem(decoder)
		if err != nil {
			return packages, fmt.Errorf("decode go list stream: %w", err)
		}

		if done {
			return packages, nil
		}

		packages = append(packages, packageFromGoList(&item))
	}
}

func emptyBlock() domain.Block {
	return domain.Block{
		File:       emptyString,
		Position:   emptyString,
		Statements: zero,
		Covered:    false,
	}
}

func finishFailedTextReport(text *textTestReport, result *domain.Coverage, fields []string) {
	text.failed = text.failed || !result.TestsFailed || len(fields) > one

	if len(fields) > one {
		flushTextReport(text, result, fields[one])
	}
}

func finishHTMLReport(
	ctx context.Context,
	file *os.File,
	request *collectionrequest.HTMLOpenRequest,
) error {
	err := writeAndCloseProfile(file, request.Profile)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	err = runGoCover(ctx, file.Name(), request)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	return nil
}

func finishProfileScan(scanner *bufio.Scanner) ([]domain.Block, error) {
	blocks, err := readProfileBlocks(scanner)
	if err != nil {
		return nil, fmt.Errorf("finish profile scan: %w", err)
	}

	err = scanner.Err()
	if err != nil {
		return nil, fmt.Errorf(errReadCoverageProfileFormat, err)
	}

	return blocks, nil
}

func finishTextReportFields(text *textTestReport, result *domain.Coverage, fields []string) {
	switch fields[zero] {
	case textFailAction:
		finishFailedTextReport(text, result, fields)
	case "ok", "?":
		flushTextReport(text, result, emptyString)
	default:
	}
}

func finishTextReportLine(text *textTestReport, result *domain.Coverage, line string) {
	if indentedReportLine(line) {
		return
	}

	fields := strings.Fields(line)

	if len(fields) == zero {
		return
	}

	finishTextReportFields(text, result, fields)
}

func firstFile(item *goListPackage, files []string) string {
	if len(files) > zero {
		return files[zero]
	}

	if len(item.TestGoFiles) == zero {
		return emptyString
	}

	return absolutePackageFile(item.Dir, item.TestGoFiles[zero])
}

func flagNeedsValue(name string) bool {
	switch name {
	case flagRace, flagAsan, flagMsan, flagTrimpath:
		return false
	default:
		return true
	}
}

func flushTextReport(text *textTestReport, result *domain.Coverage, pkg string) {
	if text.failed {
		appendTextReportFailures(text, result, pkg)
	}

	var empty textTestReport

	*text = empty
}

func goCoverCommand(ctx context.Context, profilePath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, goCommand, "tool", "cover")

	cmd.Args = append(cmd.Args, "-html="+profilePath)

	return cmd
}

func goCoverError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf(errOpenHTMLFormat, context.Cause(ctx))
	}

	return fmt.Errorf(errOpenHTMLFormat, err)
}

func goListCommand(ctx context.Context, request *ports.PackageRequest) *exec.Cmd {
	cmd := exec.CommandContext(ctx, goCommand, goListCommandName, goListJSONFlag)

	cmd.Args = append(cmd.Args, ListArgsForTestArgs(request.TestArgs)...)
	cmd.Args = append(cmd.Args, request.Patterns...)

	return cmd
}

func goListError(ctx context.Context, err error, output []byte) error {
	if ctx.Err() != nil {
		return fmt.Errorf("go list context: %w", context.Cause(ctx))
	}

	message := string(bytes.TrimSpace(output))

	if message == emptyString {
		return fmt.Errorf(errWrapJoined, errGoListFailed, err)
	}

	return fmt.Errorf("%w: %s", errors.Join(errGoListFailed, err), message)
}

func goListRawMessage(raw map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	value, ok := raw[key]

	if ok {
		return value, true
	}

	for existing := range raw {
		if strings.EqualFold(existing, key) {
			return raw[existing], true
		}
	}

	return nil, false
}

func goListWaitFailure(ctx context.Context, run *goListRun, errs *waitErrs) error {
	if errs.decode != nil {
		return fmt.Errorf(errWaitGoListDecodeFormat, joinGoListWaitError(errs.decode, errs.wait))
	}

	if errs.wait != nil {
		return fmt.Errorf(errWaitGoListFormat, goListError(ctx, errs.wait, run.stderr.Bytes()))
	}

	return nil
}

func goTestCommand(ctx context.Context, run *goTestRun) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx,
		goCommand,
		"test",
		"-count=1",
		"-covermode=atomic",
		"-coverprofile",
	)

	cmd.Args = append(cmd.Args, run.profilePath)
	cmd.Args = append(cmd.Args, run.request.TestArgs...)
	cmd.Args = append(cmd.Args, "-json")
	cmd.Args = append(cmd.Args, run.request.Patterns...)
	setGoTestOutput(cmd, run)

	return cmd
}

func goTestError(ctx context.Context, err error, output string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("go test context: %w", context.Cause(ctx))
	}

	message := strings.TrimSpace(output)

	if message == emptyString {
		return fmt.Errorf(errWrapJoined, errGoTestFailed, err)
	}

	return fmt.Errorf("%w:\n%s", errors.Join(errGoTestFailed, err), message)
}

func hasListFlagValue(lookup *listFlagLookup) bool {
	return lookup.hasValue || !flagNeedsValue(lookup.name) ||
		lookup.index+one >= len(lookup.testArgs)
}

func indentedReportLine(line string) bool {
	return strings.HasPrefix(line, spaceSeparator) || strings.HasPrefix(line, tabSeparator)
}

func invalidColon(colon int, line string) bool {
	return colon <= zero || colon == len(line)-one
}

func invalidProfileFields(fields []string) bool {
	return len(fields) != three || !validPosition(fields[zero])
}

func joinGoListWaitError(err, waitErr error) error {
	if waitErr == nil {
		return fmt.Errorf(errWaitGoListDecodeFormat, err)
	}

	return errors.Join(
		fmt.Errorf(errWaitGoListDecodeFormat, err),
		fmt.Errorf(errWaitGoListFormat, waitErr),
	)
}

func joinProfileRead(data []byte, readErr, closeErr error) ([]byte, error) {
	if readErr != nil {
		return nil, errors.Join(
			fmt.Errorf(errReadTempProfileFormat, readErr),
			wrapCloseErr(closeErr),
		)
	}

	if closeErr != nil {
		return nil, fmt.Errorf(errCloseTempProfileFormat, closeErr)
	}

	return data, nil
}

func joinWriteCloseError(file *os.File, written int, err error) error {
	return errors.Join(
		fmt.Errorf("write temporary HTML coverage input: wrote %d: %w", written, err),
		closeHTMLInputErr(file),
	)
}

func labelReportFailures(result *domain.Coverage, path string) {
	for i := range result.Failures {
		result.Failures[i].Output = fmt.Sprintf(
			"existing test report %q:\n%s", path, result.Failures[i].Output,
		)
	}
}

func listArgsForTestArg(testArgs []string, index int) (args []string, consumed int) {
	arg := testArgs[index]
	name, hasValue, ok := splitFlag(arg)

	if !ok || !listCompatibleFlag(name) {
		return nil, zero
	}

	lookup := &listFlagLookup{name: name, hasValue: hasValue, testArgs: testArgs, index: index}

	if hasListFlagValue(lookup) {
		return []string{arg}, zero
	}

	return []string{arg, testArgs[index+one]}, one
}

func listCompatibleFlag(name string) bool {
	switch name {
	case "tags", "mod", "modfile", "overlay", flagRace, flagAsan, flagMsan,
		"gcflags", "asmflags", "ldflags", "buildmode", "compiler",
		"gccgoflags", "installsuffix", flagTrimpath, "pgo", "pkgdir", "toolexec":
		return true
	default:
		return false
	}
}

func mergeJSONReport(result *domain.Coverage, capture *testCaptureState) {
	if len(capture.failed) == zero {
		return
	}

	imported := domain.Coverage{
		TestOutput:     emptyString,
		Profile:        nil,
		Blocks:         nil,
		TestsFailed:    true,
		Failures:       captureFailures(capture),
		FailedPackages: captureFailedPackages(capture, nil),
	}
	domain.MergeTestFailures(result, &imported)
}

func newTestCapture(output *CappedBuffer) *testCaptureState {
	state := &testCaptureState{
		output:      output,
		logs:        make(map[testKey][]byte),
		packageLogs: make(map[string][]byte),
		reportLogs:  make(map[string][]byte),
		reportOrder: nil,
		passed:      make(map[testKey]bool),
		order:       nil,
		failed:      nil,
		line:        nil,
		drop:        false,
	}

	return state
}

func newTestReportParser() *testReportParser {
	output := NewCappedBuffer(commandOutputLimit)

	parser := new(testReportParser)

	parser.capture = newTestCapture(&output)

	return parser
}

func openHTMLReport(ctx context.Context, request *collectionrequest.HTMLOpenRequest) (err error) {
	file, profilePath, err := createHTMLProfileFileContext(ctx)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	defer func() {
		err = errors.Join(err, removeFileContext(ctx, profilePath))
	}()

	err = finishHTMLReport(ctx, file, request)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	return nil
}

func packageFromGoList(item *goListPackage) domain.Package {
	files := make([]string, zero, len(item.GoFiles)+len(item.CgoFiles))

	files = append(files, item.GoFiles...)
	files = append(files, item.CgoFiles...)

	for index := range files {
		files[index] = absolutePackageFile(item.Dir, files[index])
	}

	return domain.Package{
		ImportPath: item.ImportPath,
		Dir:        item.Dir,
		Files:      files,
		FirstFile:  firstFile(item, files),
	}
}

func parseCovered(value string) (bool, error) {
	count, err := strconv.ParseUint(value, decimalBase, intBitSize)
	if err != nil {
		return false, fmt.Errorf("parse execution count: %w", err)
	}

	return count > zero, nil
}

func parseFailPackageLine(line string) (string, bool) {
	const failPrefix = "FAIL\t"

	if !strings.HasPrefix(line, failPrefix) {
		return emptyString, false
	}

	parts := strings.SplitN(strings.TrimPrefix(line, failPrefix), tabSeparator, two)
	pkg := strings.TrimSpace(strings.TrimSuffix(parts[zero], " [build failed]"))

	if pkg == emptyString {
		return emptyString, false
	}

	return pkg, true
}

func parseFailedPackages(output string) []string {
	lines := strings.Split(output, "\n")
	packages := make([]string, zero, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for i := range lines {
		packages = appendParsedFailPackage(packages, seen, lines[i])
	}

	return packages
}

func parseProfileBlock(line string, lineNumber int) (domain.Block, error) {
	colon := strings.LastIndexByte(line, ':')

	if invalidColon(colon, line) {
		return domain.Block{}, fmt.Errorf(errWrapLine, errMalformedProfileLine, lineNumber)
	}

	block, err := parseProfileFields(line, colon, lineNumber)
	if err != nil {
		return domain.Block{}, fmt.Errorf("parse profile block: %w", err)
	}

	return block, nil
}

func parseProfileFields(line string, colon, lineNumber int) (domain.Block, error) {
	fields := strings.Fields(line[colon+one:])

	if invalidProfileFields(fields) {
		return domain.Block{}, fmt.Errorf(errWrapLine, errMalformedProfileLine, lineNumber)
	}

	block, err := buildProfileBlock(line[:colon], fields, lineNumber)
	if err != nil {
		return domain.Block{}, fmt.Errorf("parse profile fields: %w", err)
	}

	return block, nil
}

func parseProfileLine(line string, lineNumber int) (block domain.Block, ok bool, err error) {
	line = strings.TrimSpace(line)

	if line == emptyString {
		return emptyBlock(), false, nil
	}

	block, err = parseProfileBlock(line, lineNumber)
	if err != nil {
		return domain.Block{}, true, fmt.Errorf("parse profile line: %w", err)
	}

	return block, true, nil
}

func parseReportLine(parser *testReportParser, line string) {
	var event testEvent

	if json.Unmarshal([]byte(line), &event) == nil && event.Action != emptyString {
		captureRecordEvent(parser.capture, &event)

		return
	}

	parseTextReportLine(&parser.text, &parser.result, line)
}

func parseStatementCount(value string) (int64, error) {
	statements, err := strconv.ParseInt(value, decimalBase, intBitSize)
	if err != nil {
		return zero, fmt.Errorf("parse statement count: %w", err)
	}

	if statements < zero {
		return zero, errInvalidStatementCount
	}

	return statements, nil
}

func parseTestReport(scanner *bufio.Scanner, path string) (domain.Coverage, error) {
	parser := newTestReportParser()

	scanner.Buffer(nil, maxScannerBufferSize)

	for scanner.Scan() {
		parseReportLine(parser, scanner.Text())
	}

	flushTextReport(&parser.text, &parser.result, emptyString)
	mergeJSONReport(&parser.result, parser.capture)
	labelReportFailures(&parser.result, path)

	return parser.result, errors.Join(reportScanError(scanner.Err(), path))
}

func parseTestReportInto(reader io.Reader, path string, coverage *domain.Coverage) error {
	parsed, err := parseTestReport(bufio.NewScanner(reader), path)
	if err != nil {
		return fmt.Errorf("parse test report: %w", err)
	}

	*coverage = parsed

	return nil
}

func parseTextReportLine(text *textTestReport, result *domain.Coverage, line string) {
	trimmed := strings.TrimSpace(line)
	appendTextReportOutput(text, line)

	if strings.HasPrefix(trimmed, textTestFailPrefix) {
		text.failed = true
		text.failures = append(text.failures, domain.TestFailure{
			Test: textFailureName(trimmed), ImportPath: emptyString, Output: emptyString,
		})
	}

	appendTextFailureOutput(text, line)
	finishTextReportLine(text, result, line)
}

func prepareGoList(
	ctx context.Context,
	request *ports.PackageRequest,
) (*goListRun, error) {
	cmd := goListCommand(ctx, request)
	stderr := NewCappedBuffer(commandOutputLimit)

	cmd.Stderr = &stderr

	run, err := startGoListCommand(cmd, &stderr)
	if err != nil {
		return nil, fmt.Errorf("prepare package list: %w", err)
	}

	return run, nil
}

func profileHeaderScanError(scanner *bufio.Scanner) error {
	err := scanner.Err()
	if err != nil {
		return fmt.Errorf(errReadCoverageProfileFormat, err)
	}

	return errEmptyCoverageProfile
}

func readAndCloseProfile(file *os.File) ([]byte, error) {
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()

	result, err := joinProfileRead(data, readErr, closeErr)
	if closeErr != nil {
		err = retry.Stop(err)
	}

	if err != nil {
		return nil, fmt.Errorf(errReadTempProfileFormat, err)
	}

	return result, nil
}

func readGoListObject(
	decoder *json.Decoder,
) (raw map[string]json.RawMessage, done bool, err error) {
	err = decoder.Decode(&raw)

	if errors.Is(err, io.EOF) {
		return map[string]json.RawMessage{}, true, nil
	}

	if err != nil {
		return nil, true, fmt.Errorf(errDecodeGoListItem, err)
	}

	return raw, false, nil
}

func readParsedCoverageContext(ctx context.Context, profilePath string) (domain.Coverage, error) {
	profileData, err := readTempProfileContext(ctx, profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errReadTempProfileFormat, err)
	}

	blocks, err := ParseProfile(bytes.NewReader(profileData))
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errParseCoverageProfileFormat, err)
	}

	return domain.Coverage{
		Profile:        profileData,
		Blocks:         blocks,
		FailedPackages: nil,
		TestOutput:     emptyString,
		TestsFailed:    false,
		Failures:       nil,
	}, nil
}

func readProfileBlocks(scanner *bufio.Scanner) ([]domain.Block, error) {
	blocks := make([]domain.Block, zero, initialProfileBlockCap)
	lineNumber := one

	for scanner.Scan() {
		lineNumber++

		block, ok, err := parseProfileLine(scanner.Text(), lineNumber)
		if err != nil {
			return nil, fmt.Errorf("read profile blocks: %w", err)
		}

		if ok {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

func readProfileHeader(scanner *bufio.Scanner) error {
	if !scanner.Scan() {
		return fmt.Errorf("read profile header: %w", profileHeaderScanError(scanner))
	}

	if !strings.HasPrefix(scanner.Text(), profileModePrefix) {
		return errMissingProfileMode
	}

	return nil
}

func readTempProfileAttempt(profilePath string) ([]byte, error) {
	file, err := os.Open(filepath.Clean(profilePath))
	if err != nil {
		return nil, fmt.Errorf("open temporary coverage profile: %w", err)
	}

	data, err := readAndCloseProfile(file)
	if err != nil {
		return nil, fmt.Errorf(errReadTempProfileFormat, err)
	}

	return data, nil
}

func readTempProfileContext(ctx context.Context, profilePath string) ([]byte, error) {
	var data []byte

	err := retry.Run(ctx, func() error {
		var err error

		data, err = readTempProfileAttempt(profilePath)
		if err != nil {
			return fmt.Errorf(errReadTempProfileFormat, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(errReadTempProfileFormat, err)
	}

	return data, nil
}

func readTestReportFile(ctx context.Context, path string) (domain.Coverage, error) {
	var coverage domain.Coverage

	err := retry.ReadFile(ctx, path, func(reader io.Reader) error {
		return parseTestReportInto(reader, path, &coverage)
	})
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("read report file: %w", err)
	}

	return coverage, nil
}

func removeFileContext(ctx context.Context, name string) error {
	err := retry.Run(context.WithoutCancel(ctx), func() error {
		err := os.Remove(name)
		if err != nil {
			return fmt.Errorf("remove path: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("remove temporary file: %w", err)
	}

	return nil
}

func reportScanError(err error, path string) error {
	if err != nil {
		return fmt.Errorf("parse test report %q: %w", path, err)
	}

	return nil
}

func runCommand(ctx context.Context, command func() *exec.Cmd) error {
	cmd, err := startCommand(ctx, command, (*exec.Cmd).Start)
	if err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("wait command: %w", err)
	}

	return nil
}

func runGoCover(
	ctx context.Context,
	profilePath string,
	request *collectionrequest.HTMLOpenRequest,
) error {
	err := runCommand(ctx, func() *exec.Cmd {
		cmd := goCoverCommand(ctx, profilePath)

		cmd.Stdout = request.Stdout
		cmd.Stderr = request.Stderr

		return cmd
	})
	if err != nil {
		return fmt.Errorf("run go cover: %w", goCoverError(ctx, err))
	}

	return nil
}

func runGoListAssigns(assigns []func() error) error {
	for index := range assigns {
		err := assigns[index]()
		if err != nil {
			return fmt.Errorf(errUnmarshalGoListFormat, err)
		}
	}

	return nil
}

func runGoTest(ctx context.Context, run *goTestRun) error {
	err := runCommand(ctx, func() *exec.Cmd { return goTestCommand(ctx, run) })
	if err != nil {
		return fmt.Errorf("run go test: %w", err)
	}

	return nil
}

func setGoTestOutput(cmd *exec.Cmd, run *goTestRun) {
	cmd.Stdout = run.output
	cmd.Stderr = run.output

	if run.capture != nil {
		output := testCaptureOutput(run.capture)

		cmd.Stdout = output
		cmd.Stderr = output
	}
}

func splitFlag(arg string) (name string, hasValue, ok bool) {
	if !strings.HasPrefix(arg, flagPrefix) || arg == flagPrefix {
		return emptyString, false, false
	}

	trimmed := strings.TrimLeft(arg, flagPrefix)

	if trimmed == emptyString {
		return emptyString, false, false
	}

	parts := strings.SplitN(trimmed, "=", two)

	if len(parts) > one {
		return parts[zero], true, true
	}

	return trimmed, false, true
}

func startCommand(
	ctx context.Context, command func() *exec.Cmd, start func(*exec.Cmd) error,
) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	err := retry.Run(ctx, func() error {
		cmd = command()

		err := start(cmd)
		if err != nil {
			return fmt.Errorf("start attempt: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	return cmd, nil
}

func startGoListCommand(cmd *exec.Cmd, stderr *CappedBuffer) (*goListRun, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare go list: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start go list: %w", err)
	}

	return &goListRun{cmd: cmd, stdout: stdout, stderr: stderr}, nil
}

func startPackageList(
	ctx context.Context,
	request *ports.PackageRequest,
) (*goListRun, error) {
	var run *goListRun

	err := retry.Run(ctx, func() error {
		var err error

		run, err = prepareGoList(ctx, request)
		if err != nil {
			return fmt.Errorf("prepare list attempt: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("start package list: %w", err)
	}

	return run, nil
}

func testCaptureOutput(capture *testCaptureState) testCapture {
	return func(data []byte) int { return writeTestCapture(capture, data) }
}

func textFailureName(line string) string {
	fields := strings.Fields(strings.TrimPrefix(line, textTestFailPrefix))

	if len(fields) == zero {
		return emptyString
	}

	return fields[zero]
}

func truncatedSuffixIndex(limit, length, mode int) int {
	if mode != zero && limit+length >= zero {
		return one
	}

	return zero
}

func unmarshalGoListString(raw map[string]json.RawMessage, key string, target *string) error {
	value, ok := goListRawMessage(raw, key)

	if !ok {
		*target = emptyString

		return nil
	}

	err := json.Unmarshal(value, target)
	if err != nil {
		return fmt.Errorf(errUnmarshalGoListFieldFormat, key, err)
	}

	return nil
}

func unmarshalGoListStrings(raw map[string]json.RawMessage, key string, target *[]string) error {
	value, ok := goListRawMessage(raw, key)

	if !ok {
		*target = nil

		return nil
	}

	err := json.Unmarshal(value, target)
	if err != nil {
		return fmt.Errorf(errUnmarshalGoListFieldFormat, key, err)
	}

	return nil
}

func validPosition(value string) bool {
	comma := strings.IndexByte(value, ',')

	return comma > zero && comma < len(value)-one && strings.Contains(value[:comma], positionDot) &&
		strings.Contains(value[comma+one:], positionDot)
}

func waitGoList(ctx context.Context, run *goListRun) ([]domain.Package, error) {
	packages, err := decodeGoListOutput(run.stdout)
	waitErr := run.cmd.Wait()

	err = goListWaitFailure(ctx, run, &waitErrs{decode: err, wait: waitErr})
	if err != nil {
		return nil, fmt.Errorf(errWaitGoListFormat, err)
	}

	return packages, nil
}

func wrapCloseErr(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf(errCloseTempProfileFormat, err)
}

func writeAndCloseProfile(file *os.File, profile []byte) error {
	err := writeAndCloseProfileWith(file, profile, closeFile)
	if err != nil {
		return fmt.Errorf("write and close profile: %w", err)
	}

	return nil
}

func writeAndCloseProfileWith(
	file *os.File,
	profile []byte,
	closer func(*os.File) error,
) error {
	written, err := file.Write(profile)
	if err != nil {
		return fmt.Errorf(
			"write temporary HTML coverage input: %w",
			joinWriteCloseError(file, written, err),
		)
	}

	err = closeHTMLInputWith(file, closer)
	if err != nil {
		return fmt.Errorf(errCloseHTMLInputFormat, err)
	}

	return nil
}

func writeCapped(buf *cappedWrite, data []byte) (int, error) {
	if *buf.truncated || buf.limit <= zero || buf.buffer.Len() >= buf.limit {
		*buf.truncated = true

		return len(data), nil
	}

	written, err := writeCappedRemaining(buf, data)
	if err != nil {
		return written, fmt.Errorf(errWriteCappedBuffer, err)
	}

	return written, nil
}

func writeCappedAll(buf *cappedWrite, data []byte) (int, error) {
	written, err := buf.buffer.Write(data)
	if err != nil {
		return written, fmt.Errorf("write capped buffer data: %w", err)
	}

	return len(data), nil
}

func writeCappedBytes(buf *cappedWrite, data []byte) int {
	written, err := writeCapped(buf, data)
	if err != nil {
		return written
	}

	return written
}

func writeCappedOverflow(buf *cappedWrite, data []byte, remaining int) (int, error) {
	written, err := buf.buffer.Write(data[:remaining])
	if err != nil {
		return written, fmt.Errorf("write capped buffer limit: %w", err)
	}

	*buf.truncated = true

	return len(data), nil
}

func writeCappedRemaining(buf *cappedWrite, data []byte) (int, error) {
	remaining := buf.limit - buf.buffer.Len()

	var written int

	var err error

	if len(data) > remaining {
		written, err = writeCappedOverflow(buf, data, remaining)
	}

	if len(data) <= remaining {
		written, err = writeCappedAll(buf, data)
	}

	if err != nil {
		return written, fmt.Errorf(errWriteCappedBuffer, err)
	}

	return written, nil
}

func writeTestCapture(capture *testCaptureState, data []byte) int {
	for i := range data {
		captureConsume(capture, data[i])
	}

	return len(data)
}

// Bytes returns the buffered output as bytes.
func (buf *CappedBuffer) Bytes() []byte {
	mode := len(map[bool]struct{}{false: {}, buf.truncated: {}}) - one

	return append(
		buf.buffer.Bytes(),
		cappedSuffixes()[truncatedSuffixIndex(buf.limit, buf.buffer.Len(), mode)]...,
	)
}

// Collect runs go test and parses its coverage profile.
func (*Adapter) Collect(
	ctx context.Context,
	request *ports.CoverageRequest,
) (coverage domain.Coverage, err error) {
	profilePath, err := createCoverageProfile(ctx)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectCoverageFormat, err)
	}

	defer func() {
		err = errors.Join(err, removeFileContext(ctx, profilePath))
	}()

	coverage, err = collectFromProfile(ctx, profilePath, request)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectCoverageFormat, err)
	}

	return coverage, nil
}

// List returns Go package metadata for the requested package patterns.
func (*Adapter) List(
	ctx context.Context,
	request *ports.PackageRequest,
) ([]domain.Package, error) {
	run, err := startPackageList(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("list packages: start go list: %w", err)
	}

	packages, err := waitGoList(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}

	return packages, nil
}

// Open renders a coverage profile with go tool cover.
func (*Adapter) Open(ctx context.Context, request *collectionrequest.HTMLOpenRequest) error {
	if len(request.Profile) == zero {
		return errEmptyCoverageProfile
	}

	err := openHTMLReport(ctx, request)
	if err != nil {
		return fmt.Errorf("open coverage report: %w", err)
	}

	return nil
}

// String returns the captured output, with a notice when the byte limit was hit.
func (buf *CappedBuffer) String() string {
	mode := len(map[bool]struct{}{false: {}, buf.truncated: {}}) - one

	return buf.buffer.String() + cappedSuffixes()[truncatedSuffixIndex(
		buf.limit,
		buf.buffer.Len(),
		mode,
	)]
}

// Write appends data until the buffer reaches its configured byte limit.
func (buf *CappedBuffer) Write(data []byte) (int, error) {
	return writeCappedBytes(&cappedWrite{
		buffer:    buf.buffer,
		truncated: &buf.truncated,
		limit:     buf.limit,
	}, data), nil
}

// Write decodes test events and captures their output without returning write errors.
func (capture testCapture) Write(data []byte) (int, error) {
	return capture(data), nil
}
