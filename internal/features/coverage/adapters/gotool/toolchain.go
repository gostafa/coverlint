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
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
)

const commandOutputLimit = 1 << 20

type Adapter struct {
	goBinary string
}

func New() *Adapter {
	return &Adapter{goBinary: "go"}
}

func (a *Adapter) Collect(ctx context.Context, request ports.CoverageRequest) (domain.Coverage, error) {
	profile, err := os.CreateTemp("", "coverlint-*.coverprofile")
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("create temporary coverage profile: %w", err)
	}
	profilePath := profile.Name()
	if err := profile.Close(); err != nil {
		_ = os.Remove(profilePath)
		return domain.Coverage{}, fmt.Errorf("close temporary coverage profile: %w", err)
	}
	defer os.Remove(profilePath)

	arguments := []string{
		"test",
		"-count=1",
		"-covermode=atomic",
		"-coverprofile=" + profilePath,
	}
	arguments = append(arguments, request.TestArgs...)
	arguments = append(arguments, request.Patterns...)

	cmd := exec.CommandContext(ctx, a.goBinary, arguments...)
	output := newCappedBuffer(commandOutputLimit)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return domain.Coverage{}, ctx.Err()
		}
		message := strings.TrimSpace(output.String())
		if message == "" {
			return domain.Coverage{}, fmt.Errorf("go test failed: %w", err)
		}
		return domain.Coverage{}, fmt.Errorf("go test failed:\n%s", message)
	}

	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return domain.Coverage{}, fmt.Errorf("read temporary coverage profile: %w", err)
	}
	blocks, err := parseProfile(bytes.NewReader(profileData))
	if err != nil {
		return domain.Coverage{}, err
	}
	return domain.Coverage{Profile: profileData, Blocks: blocks}, nil
}

func (a *Adapter) List(ctx context.Context, request ports.PackageRequest) ([]domain.Package, error) {
	arguments := []string{"list", "-json"}
	arguments = append(arguments, listArgsForTestArgs(request.TestArgs)...)
	arguments = append(arguments, request.Patterns...)
	cmd := exec.CommandContext(ctx, a.goBinary, arguments...)
	stderr := newCappedBuffer(commandOutputLimit)
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare go list: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start go list: %w", err)
	}

	decoder := json.NewDecoder(stdout)
	packages := make([]domain.Package, 0, 32)
	for {
		var item goListPackage
		if err := decoder.Decode(&item); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = cmd.Wait()
			return nil, fmt.Errorf("decode go list output: %w", err)
		}

		files := make([]string, 0, len(item.GoFiles)+len(item.CgoFiles))
		files = append(files, item.GoFiles...)
		files = append(files, item.CgoFiles...)
		for i, filename := range files {
			if !filepath.IsAbs(filename) {
				files[i] = filepath.Join(item.Dir, filename)
			}
		}

		firstFile := ""
		if len(files) > 0 {
			firstFile = files[0]
		} else if len(item.TestGoFiles) > 0 {
			firstFile = item.TestGoFiles[0]
			if !filepath.IsAbs(firstFile) {
				firstFile = filepath.Join(item.Dir, firstFile)
			}
		}

		packages = append(packages, domain.Package{
			ImportPath: item.ImportPath,
			Dir:        item.Dir,
			Files:      files,
			FirstFile:  firstFile,
		})
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		message := string(bytes.TrimSpace(stderr.Bytes()))
		if message == "" {
			return nil, fmt.Errorf("go list failed: %w", err)
		}
		return nil, fmt.Errorf("go list failed: %s", message)
	}
	return packages, nil
}

func listArgsForTestArgs(testArgs []string) []string {
	listArgs := make([]string, 0, len(testArgs))
	for i := 0; i < len(testArgs); i++ {
		arg := testArgs[i]
		name, hasValue, ok := splitFlag(arg)
		if !ok || !listCompatibleFlag(name) {
			continue
		}
		listArgs = append(listArgs, arg)
		if !hasValue && flagNeedsValue(name) && i+1 < len(testArgs) {
			i++
			listArgs = append(listArgs, testArgs[i])
		}
	}
	return listArgs
}

func splitFlag(arg string) (name string, hasValue bool, ok bool) {
	if !strings.HasPrefix(arg, "-") || arg == "-" {
		return "", false, false
	}
	trimmed := strings.TrimLeft(arg, "-")
	if trimmed == "" {
		return "", false, false
	}
	if index := strings.IndexByte(trimmed, '='); index >= 0 {
		return trimmed[:index], true, true
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

func (a *Adapter) Open(ctx context.Context, profile []byte, stdout, stderr io.Writer) error {
	if len(profile) == 0 {
		return fmt.Errorf("coverage profile is empty")
	}

	file, err := os.CreateTemp("", "coverlint-web-*.coverprofile")
	if err != nil {
		return fmt.Errorf("create temporary HTML coverage input: %w", err)
	}
	profilePath := file.Name()
	defer os.Remove(profilePath)

	if _, err := file.Write(profile); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary HTML coverage input: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary HTML coverage input: %w", err)
	}

	cmd := exec.CommandContext(ctx, a.goBinary, "tool", "cover", "-html="+profilePath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("open HTML coverage report: %w", ctx.Err())
		}
		return fmt.Errorf("open HTML coverage report: %w", err)
	}
	return nil
}

type goListPackage struct {
	ImportPath  string
	Dir         string
	GoFiles     []string
	CgoFiles    []string
	TestGoFiles []string
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func newCappedBuffer(limit int) cappedBuffer {
	return cappedBuffer{limit: limit}
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
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

func (b *cappedBuffer) String() string {
	if !b.truncated {
		return b.buffer.String()
	}
	return b.buffer.String() + "\n... output truncated by coverlint ..."
}

func (b *cappedBuffer) Bytes() []byte {
	return []byte(b.String())
}
