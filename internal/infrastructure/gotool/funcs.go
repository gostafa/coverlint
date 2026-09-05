// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports/outbound"
)

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

// Collect runs go test and parses its coverage profile.
func (*Adapter) Collect(
	ctx context.Context,
	request *outbound.CoverageRequest,
) (coverage domain.Coverage, err error) {
	profilePath, err := createTempProfile()
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectCoverageFormat, err)
	}

	defer func() {
		err = errors.Join(err, removeFile(profilePath))
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
	request *outbound.PackageRequest,
) ([]domain.Package, error) {
	cmd := goListCommand(ctx, request)
	stderr := NewCappedBuffer(commandOutputLimit)

	cmd.Stderr = &stderr

	packages, err := runGoList(ctx, cmd, &stderr)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}

	return packages, nil
}

// Open renders a coverage profile with go tool cover.
func (*Adapter) Open(ctx context.Context, request *outbound.HTMLOpenRequest) error {
	if len(request.Profile) == zero {
		return errEmptyCoverageProfile
	}

	err := openHTMLReport(ctx, request)
	if err != nil {
		return fmt.Errorf("open coverage report: %w", err)
	}

	return nil
}

// Bytes returns the buffered output as bytes.
func (buf *CappedBuffer) Bytes() []byte {
	mode := len(map[bool]struct{}{false: {}, buf.truncated: {}}) - one

	return append(
		buf.buffer.Bytes(),
		cappedSuffixes()[truncatedSuffixIndex(buf.limit, buf.buffer.Len(), mode)]...,
	)
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

func cappedSuffixes() [two]string {
	return [two]string{emptyString, truncationSuffix}
}

func truncatedSuffixIndex(limit, length, mode int) int {
	if mode != zero && limit+length >= zero {
		return one
	}

	return zero
}

func writeCappedBytes(buf *cappedWrite, data []byte) int {
	written, err := writeCapped(buf, data)
	if err != nil {
		return written
	}

	return written
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

func writeCappedAll(buf *cappedWrite, data []byte) (int, error) {
	written, err := buf.buffer.Write(data)
	if err != nil {
		return written, fmt.Errorf("write capped buffer data: %w", err)
	}

	return len(data), nil
}

func writeCappedOverflow(buf *cappedWrite, data []byte, remaining int) (int, error) {
	written, err := buf.buffer.Write(data[:remaining])
	if err != nil {
		return written, fmt.Errorf("write capped buffer limit: %w", err)
	}

	*buf.truncated = true

	return len(data), nil
}

func absolutePackageFile(dir, filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}

	return filepath.Join(dir, filename)
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

func runGoListAssigns(assigns []func() error) error {
	for index := range assigns {
		err := assigns[index]()
		if err != nil {
			return fmt.Errorf(errUnmarshalGoListFormat, err)
		}
	}

	return nil
}

func closeCreatedTempProfile(profile *os.File) (string, error) {
	profilePath, err := closeCreatedTempProfileWith(profile, closeFile)
	if err != nil {
		return emptyString, fmt.Errorf(errCloseCreatedTempProfile, err)
	}

	return profilePath, nil
}

func closeCreatedTempProfileWith(
	profile *os.File,
	closer func(*os.File) error,
) (string, error) {
	profilePath := profile.Name()

	err := closer(profile)
	if err != nil {
		return emptyString, fmt.Errorf(
			errCloseCreatedTempProfile,
			closeTempProfileError(profilePath, err),
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

func closeHTMLInputWith(file *os.File, closer func(*os.File) error) error {
	err := closer(file)
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

func closeTempProfileError(profilePath string, err error) error {
	return errors.Join(
		fmt.Errorf(errCloseTempProfileFormat, err),
		removeFile(profilePath),
	)
}

func collectFromProfile(
	ctx context.Context,
	profilePath string,
	request *outbound.CoverageRequest,
) (domain.Coverage, error) {
	output := NewCappedBuffer(commandOutputLimit)
	run := &goTestRun{profilePath: profilePath, request: request, output: &output}

	coverage, err := collectGoTestResult(ctx, run, runGoTest(ctx, run))
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, err)
	}

	return coverage, nil
}

func annotateFailedCoverage(coverage *domain.Coverage, output string) domain.Coverage {
	coverage.FailedPackages = parseFailedPackages(output)
	coverage.TestOutput = strings.TrimSpace(output)
	coverage.TestsFailed = true

	return *coverage
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

	coverage, err := readParsedCoverage(run.profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, hardErr)
	}

	return annotateFailedCoverage(&coverage, output), nil
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
		coverage, err = collectPassedGoTest(run)
	}

	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errWrap, err)
	}

	return coverage, nil
}

func collectPassedGoTest(run *goTestRun) (domain.Coverage, error) {
	coverage, err := readParsedCoverage(run.profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf(errCollectFromProfileFormat, err)
	}

	coverage.TestOutput = strings.TrimSpace(run.output.String())

	return coverage, nil
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

func parseFailPackageLine(line string) (string, bool) {
	const failPrefix = "FAIL\t"

	if !strings.HasPrefix(line, failPrefix) {
		return emptyString, false
	}

	parts := strings.SplitN(strings.TrimPrefix(line, failPrefix), "\t", two)
	pkg := strings.TrimSpace(strings.TrimSuffix(parts[zero], " [build failed]"))

	if pkg == emptyString {
		return emptyString, false
	}

	return pkg, true
}

func createHTMLProfileFile() (*os.File, string, error) {
	file, err := os.CreateTemp(emptyString, "coverlint-web-*.coverprofile")
	if err != nil {
		return nil, emptyString, fmt.Errorf("create temporary HTML coverage input: %w", err)
	}

	profilePath := file.Name()

	return file, profilePath, nil
}

func createTempProfile() (string, error) {
	profilePath, err := createTempProfileWith(closeFile)
	if err != nil {
		return emptyString, fmt.Errorf(errCreateTempProfileFormat, err)
	}

	return profilePath, nil
}

func createTempProfileWith(closer func(*os.File) error) (string, error) {
	profile, err := os.CreateTemp(emptyString, tempProfilePattern)
	if err != nil {
		return emptyString, fmt.Errorf(errCreateTempProfileFormat, err)
	}

	profilePath, err := closeCreatedTempProfileWith(profile, closer)
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
		return goListPackage{}, true, nil
	}

	item, err = assignGoListPackage(raw)
	if err != nil {
		return goListPackage{}, false, fmt.Errorf(errDecodeGoListItem, err)
	}

	return item, false, nil
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

func goCoverCommand(ctx context.Context, profilePath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, goCommand, "tool", "cover")

	cmd.Args = append(cmd.Args, "-html="+profilePath)

	return cmd
}

func goCoverError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf(errOpenHTMLFormat, ctx.Err())
	}

	return fmt.Errorf(errOpenHTMLFormat, err)
}

func goListCommand(ctx context.Context, request *outbound.PackageRequest) *exec.Cmd {
	cmd := exec.CommandContext(ctx, goCommand, goListCommandName, goListJSONFlag)

	cmd.Args = append(cmd.Args, ListArgsForTestArgs(request.TestArgs)...)
	cmd.Args = append(cmd.Args, request.Patterns...)

	return cmd
}

func goListError(ctx context.Context, err error, output []byte) error {
	if ctx.Err() != nil {
		return fmt.Errorf("go list context: %w", ctx.Err())
	}

	message := string(bytes.TrimSpace(output))

	if message == emptyString {
		return fmt.Errorf(errWrapJoined, errGoListFailed, err)
	}

	return fmt.Errorf("%w: %s", errGoListFailed, message)
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
	cmd.Args = append(cmd.Args, run.request.Patterns...)
	cmd.Stdout = run.output
	cmd.Stderr = run.output

	return cmd
}

func goTestError(ctx context.Context, err error, output string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("go test context: %w", ctx.Err())
	}

	message := strings.TrimSpace(output)

	if message == emptyString {
		return fmt.Errorf(errWrapJoined, errGoTestFailed, err)
	}

	return fmt.Errorf("%w:\n%s", errGoTestFailed, message)
}

func hasListFlagValue(lookup *listFlagLookup) bool {
	return lookup.hasValue || !flagNeedsValue(lookup.name) ||
		lookup.index+one >= len(lookup.testArgs)
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

func openHTMLReport(ctx context.Context, request *outbound.HTMLOpenRequest) (err error) {
	file, profilePath, err := createHTMLProfileFile()
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	defer func() {
		err = errors.Join(err, removeFile(profilePath))
	}()

	err = finishHTMLReport(ctx, file, request)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	return nil
}

func finishHTMLReport(ctx context.Context, file *os.File, request *outbound.HTMLOpenRequest) error {
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

func readAndCloseProfile(file *os.File) ([]byte, error) {
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()

	result, err := joinProfileRead(data, readErr, closeErr)
	if err != nil {
		return nil, fmt.Errorf(errReadTempProfileFormat, err)
	}

	return result, nil
}

func readParsedCoverage(profilePath string) (domain.Coverage, error) {
	profileData, err := readTempProfile(profilePath)
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
	}, nil
}

func readTempProfile(profilePath string) ([]byte, error) {
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

func removeFile(name string) error {
	err := os.Remove(name)
	if err != nil {
		return fmt.Errorf("remove temporary file: %w", err)
	}

	return nil
}

func runGoCover(ctx context.Context, profilePath string, request *outbound.HTMLOpenRequest) error {
	cmd := goCoverCommand(ctx, profilePath)

	cmd.Stdout = request.Stdout
	cmd.Stderr = request.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("run go cover: %w", goCoverError(ctx, err))
	}

	return nil
}

func runGoList(ctx context.Context, cmd *exec.Cmd, stderr *CappedBuffer) ([]domain.Package, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare go list: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start go list: %w", err)
	}

	packages, err := waitGoList(ctx, &goListRun{cmd: cmd, stdout: stdout, stderr: stderr})
	if err != nil {
		return nil, fmt.Errorf("run go list: %w", err)
	}

	return packages, nil
}

func runGoTest(ctx context.Context, run *goTestRun) error {
	err := goTestCommand(ctx, run).Run()
	if err != nil {
		return fmt.Errorf("run go test: %w", err)
	}

	return nil
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

func waitGoList(ctx context.Context, run *goListRun) ([]domain.Package, error) {
	packages, err := decodeGoListOutput(run.stdout)
	waitErr := run.cmd.Wait()

	err = goListWaitFailure(ctx, run, &waitErrs{decode: err, wait: waitErr})
	if err != nil {
		return nil, fmt.Errorf(errWaitGoListFormat, err)
	}

	return packages, nil
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
