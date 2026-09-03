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

const (
	commandOutputLimit = 1 << 20
	initialPackageCap  = 32
)

var (
	errGoTestFailed        = errors.New("go test failed")
	errGoListFailed        = errors.New("go list failed")
	errEmptyCoverageReport = errors.New("coverage profile is empty")
)

// Adapter runs Go toolchain coverage commands.
type Adapter struct{}

// New creates a Go toolchain coverage adapter.
func New() *Adapter {
	return &Adapter{}
}

// Collect runs go test and parses its coverage profile.
func (a *Adapter) Collect(
	ctx context.Context,
	request outbound.CoverageRequest,
) (domain.Coverage, error) {

	profilePath, err := createTempProfile()
	if err != nil {
		return domain.Coverage{}, err
	}

	defer removeFile(profilePath)

	output := NewCappedBuffer(commandOutputLimit)

	err = a.runGoTest(ctx, profilePath, request, &output)
	if err != nil {
		return domain.Coverage{}, goTestError(ctx, err, output.String())
	}

	profileData, err := readTempProfile(profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("read temporary coverage profile: %w", err)
	}

	blocks, err := ParseProfile(bytes.NewReader(profileData))
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("parse coverage profile: %w", err)
	}

	return domain.Coverage{Profile: profileData, Blocks: blocks}, nil
}

func createTempProfile() (string, error) {
	profile, err := os.CreateTemp("", "coverlint-*.coverprofile")
	if err != nil {
		return "", fmt.Errorf("create temporary coverage profile: %w", err)
	}

	profilePath := profile.Name()

	err = profile.Close()
	if err != nil {
		return "", closeTempProfileError(profilePath, err)
	}

	return profilePath, nil
}

func closeTempProfileError(profilePath string, err error) error {
	removeFile(profilePath)

	return fmt.Errorf("close temporary coverage profile: %w", err)
}

func goTestError(ctx context.Context, err error, output string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("go test context: %w", ctx.Err())
	}

	message := strings.TrimSpace(output)

	if message == "" {
		return fmt.Errorf("%w: %w", errGoTestFailed, err)
	}

	return fmt.Errorf("%w:\n%s", errGoTestFailed, message)
}

// List returns Go package metadata for the requested package patterns.
func (a *Adapter) List(
	ctx context.Context,
	request outbound.PackageRequest,
) ([]domain.Package, error) {

	cmd := goListCommand(ctx, request)
	stderr := NewCappedBuffer(commandOutputLimit)

	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare go list: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start go list: %w", err)
	}

	packages, err := decodeGoListOutput(stdout)
	if err != nil {
		_ = cmd.Wait()

		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, goListError(ctx, err, stderr.Bytes())
	}

	return packages, nil
}

func decodeGoListOutput(stdout io.Reader) ([]domain.Package, error) {
	decoder := json.NewDecoder(stdout)
	packages := make([]domain.Package, 0, initialPackageCap)

	for {
		var item goListPackage

		err := decoder.Decode(&item)
		if err != nil {
			return packages, decodeGoListError(err)
		}

		packages = append(packages, packageFromGoList(item))
	}
}

func decodeGoListError(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}

	return fmt.Errorf("decode go list output: %w", err)
}

func goListError(ctx context.Context, err error, output []byte) error {
	if ctx.Err() != nil {
		return fmt.Errorf("go list context: %w", ctx.Err())
	}

	message := string(bytes.TrimSpace(output))

	if message == "" {
		return fmt.Errorf("%w: %w", errGoListFailed, err)
	}

	return fmt.Errorf("%w: %s", errGoListFailed, message)
}

// ListArgsForTestArgs returns go list-compatible build flags from go test flags.
func ListArgsForTestArgs(testArgs []string) []string {
	listArgs := make([]string, 0, len(testArgs))

	for index := 0; index < len(testArgs); index++ {
		args, consumed := listArgsForTestArg(testArgs, index)

		index += consumed

		listArgs = append(listArgs, args...)
	}

	return listArgs
}

func listArgsForTestArg(testArgs []string, index int) ([]string, int) {
	arg := testArgs[index]

	name, hasValue, ok := splitFlag(arg)

	if !ok {
		return nil, 0
	}

	if !listCompatibleFlag(name) {
		return nil, 0
	}

	if hasListFlagValue(name, hasValue, testArgs, index) {
		return []string{arg}, 0
	}

	return []string{arg, testArgs[index+1]}, 1
}

func hasListFlagValue(name string, hasValue bool, testArgs []string, index int) bool {
	return hasValue || !flagNeedsValue(name) || index+1 >= len(testArgs)
}

func splitFlag(arg string) (string, bool, bool) {
	if !strings.HasPrefix(arg, "-") || arg == "-" {
		return "", false, false
	}

	trimmed := strings.TrimLeft(arg, "-")

	if trimmed == "" {
		return "", false, false
	}

	if before, _, ok := strings.Cut(trimmed, "="); ok {
		return before, true, true
	}

	return trimmed, false, true
}

func listCompatibleFlag(name string) bool {
	switch name {
	case "tags", "mod", "modfile", "overlay", "race", "asan", "msan",
		"gcflags", "asmflags", "ldflags", "buildmode", "compiler",
		"gccgoflags", "installsuffix", "trimpath", "pgo", "pkgdir", "toolexec":
		return true
	default:
		return false
	}
}

func flagNeedsValue(name string) bool {
	switch name {
	case "race", "asan", "msan", "trimpath":
		return false
	default:
		return true
	}
}

// Open renders a coverage profile with go tool cover.
func (a *Adapter) Open(ctx context.Context, profile []byte, stdout, stderr io.Writer) error {
	if len(profile) == 0 {
		return errEmptyCoverageReport
	}

	file, err := os.CreateTemp("", "coverlint-web-*.coverprofile")
	if err != nil {
		return fmt.Errorf("create temporary HTML coverage input: %w", err)
	}

	profilePath := file.Name()

	defer removeFile(profilePath)

	err = writeAndCloseProfile(file, profile)
	if err != nil {
		return err
	}

	return runGoCover(ctx, profilePath, stdout, stderr)
}

func writeAndCloseProfile(file *os.File, profile []byte) error {
	_, err := file.Write(profile)
	if err != nil {
		_ = file.Close()

		return fmt.Errorf("write temporary HTML coverage input: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("close temporary HTML coverage input: %w", err)
	}

	return nil
}

func runGoCover(ctx context.Context, profilePath string, stdout, stderr io.Writer) error {
	cmd := goCoverCommand(ctx, profilePath)

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return goCoverError(ctx, err)
	}

	return nil
}

func goCoverError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("open HTML coverage report: %w", ctx.Err())
	}

	return fmt.Errorf("open HTML coverage report: %w", err)
}

func removeFile(name string) {
	_ = os.Remove(name)
}

type goListPackage struct {
	ImportPath  string   `json:"importPath"`
	Dir         string   `json:"dir"`
	GoFiles     []string `json:"goFiles"`
	CgoFiles    []string `json:"cgoFiles"`
	TestGoFiles []string `json:"testGoFiles"`
}

// CappedBuffer stores command output up to a byte limit.
type CappedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

// NewCappedBuffer creates a capped output buffer.
func NewCappedBuffer(limit int) CappedBuffer {
	return CappedBuffer{
		buffer:    bytes.Buffer{},
		limit:     limit,
		truncated: false,
	}
}

func (b *CappedBuffer) Write(data []byte) (int, error) {
	if b.limit <= 0 {
		b.truncated = true

		return len(data), nil
	}

	remaining := b.limit - b.buffer.Len()

	if remaining <= 0 {
		b.truncated = true

		return len(data), nil
	}

	if len(data) > remaining {
		_, _ = b.buffer.Write(data[:remaining])
		b.truncated = true

		return len(data), nil
	}

	_, _ = b.buffer.Write(data)

	return len(data), nil
}

func (b *CappedBuffer) String() string {
	if !b.truncated {
		return b.buffer.String()
	}

	return b.buffer.String() + "\n... output truncated by coverlint ..."
}

// Bytes returns the buffered output as bytes.
func (b *CappedBuffer) Bytes() []byte {
	return []byte(b.String())
}

func (a *Adapter) runGoTest(
	ctx context.Context,
	profilePath string,
	request outbound.CoverageRequest,
	output *CappedBuffer,
) error {

	cmd := exec.CommandContext(
		ctx,
		"go",
		"test",
		"-count=1",
		"-covermode=atomic",
		"-coverprofile",
	)

	cmd.Args = append(cmd.Args, profilePath)
	cmd.Args = append(cmd.Args, request.TestArgs...)
	cmd.Args = append(cmd.Args, request.Patterns...)
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("run go test: %w", err)
	}

	return nil
}

func goListCommand(ctx context.Context, request outbound.PackageRequest) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", "list", "-json")

	cmd.Args = append(cmd.Args, ListArgsForTestArgs(request.TestArgs)...)
	cmd.Args = append(cmd.Args, request.Patterns...)

	return cmd
}

func goCoverCommand(ctx context.Context, profilePath string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", "tool", "cover")

	cmd.Args = append(cmd.Args, "-html="+profilePath)

	return cmd
}

func packageFromGoList(item goListPackage) domain.Package {
	files := make([]string, 0, len(item.GoFiles)+len(item.CgoFiles))

	files = append(files, item.GoFiles...)

	files = append(files, item.CgoFiles...)

	for index, filename := range files {
		if !filepath.IsAbs(filename) {
			files[index] = filepath.Join(item.Dir, filename)
		}
	}

	return domain.Package{
		ImportPath: item.ImportPath,
		Dir:        item.Dir,
		Files:      files,
		FirstFile:  firstFile(item, files),
	}
}

func firstFile(item goListPackage, files []string) string {
	if len(files) > 0 {
		return files[0]
	}

	if len(item.TestGoFiles) == 0 {
		return ""
	}

	first := item.TestGoFiles[0]

	if !filepath.IsAbs(first) {
		first = filepath.Join(item.Dir, first)
	}

	return first
}

func readTempProfile(profilePath string) ([]byte, error) {
	file, err := os.Open(filepath.Clean(profilePath))
	if err != nil {
		return nil, fmt.Errorf("open temporary coverage profile: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read temporary coverage profile: %w", err)
	}

	return data, nil
}
