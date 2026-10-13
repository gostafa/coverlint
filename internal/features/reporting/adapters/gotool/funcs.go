// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package gotool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/gostafa/coverlint/internal/features/reporting/ports"
	"github.com/gostafa/coverlint/internal/infrastructure/failure"
	"github.com/gostafa/coverlint/internal/infrastructure/process"
	"github.com/gostafa/coverlint/internal/infrastructure/retry"
)

// New constructs an HTML reporting adapter.
func New() *Adapter { return new(Adapter) }

func closeHTMLInput(file *os.File) error {
	err := closeHTMLInputWith(file, process.CloseFile)
	if err != nil {
		return fmt.Errorf(errCloseHTMLInputFormat, err)
	}

	return nil
}

func closeHTMLInputErr(file *os.File) error {
	err := closeHTMLInputErrWith(file, process.CloseFile)
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

func createHTMLProfileFileContext(ctx context.Context) (*os.File, string, error) {
	file, err := retry.CreateTemp(ctx, "coverlint-web-*.coverprofile")
	if err != nil {
		return nil, emptyString, fmt.Errorf("create temporary HTML coverage input: %w", err)
	}

	profilePath := file.Name()

	return file, profilePath, nil
}

func finishHTMLReport(
	ctx context.Context,
	file *os.File,
	request *ports.HTMLOpenRequest,
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

func joinWriteCloseError(file *os.File, written int, err error) error {
	return errors.Join(
		fmt.Errorf("write temporary HTML coverage input: wrote %d: %w", written, err),
		closeHTMLInputErr(file),
	)
}

func openHTMLReport(ctx context.Context, request *ports.HTMLOpenRequest) (err error) {
	file, profilePath, err := createHTMLProfileFileContext(ctx)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	defer func() {
		err = errors.Join(err, process.RemoveFile(ctx, profilePath))
	}()

	err = finishHTMLReport(ctx, file, request)
	if err != nil {
		return fmt.Errorf(errOpenHTMLFormat, err)
	}

	return nil
}

func openProfile(ctx context.Context, request *ports.HTMLOpenRequest) error {
	if len(request.Profile) == zero {
		return errEmptyCoverageProfile
	}

	err := openHTMLReport(ctx, request)
	if err != nil {
		return fmt.Errorf("open coverage report: %w", err)
	}

	return nil
}

func runGoCover(
	ctx context.Context,
	profilePath string,
	request *ports.HTMLOpenRequest,
) error {
	err := process.RunCommand(ctx, func() *exec.Cmd {
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

func writeAndCloseProfile(file *os.File, profile []byte) error {
	err := writeAndCloseProfileWith(file, profile, process.CloseFile)
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

// Open renders one HTML profile and publishes stable reporting failures.
func (*Adapter) Open(ctx context.Context, request *ports.HTMLOpenRequest) error {
	return errors.Join(failure.Translate(openProfile(ctx, request), ports.ErrReporting))
}
