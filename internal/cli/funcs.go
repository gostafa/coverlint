// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/reporting/domain"
)

// Run executes coverlint with args and returns its process exit code.
func Run(args []string) int {
	return run(args, defaultStdout(), defaultStderr())
}

func defaultStdout() io.Writer { return writerFromOS("stdout") }
func defaultStderr() io.Writer { return writerFromOS("stderr") }

func run(args []string, stdout, stderr io.Writer) int {
	var opts options

	flagSet := newFlagSet(stderr, &opts)

	var usageErr error

	flagSet.Usage = func() { usageErr = printUsage(stderr, flagSet) }

	if exitCode, ok := parseFlags(flagSet, args, &usageErr); ok {
		return exitCode
	}

	if opts.showVersion {
		return printVersion(stdout)
	}

	return runCoverage(opts, flagSet.Args(), stdout, stderr)
}

func parseFlags(flagSet *flag.FlagSet, args []string, usageErr *error) (int, bool) {
	err := flagSet.Parse(args)
	if err == nil {
		return 0, false
	}

	if *usageErr != nil {
		return usageExitCode, true
	}

	if errors.Is(err, flag.ErrHelp) {
		return 0, true
	}

	return usageExitCode, true
}

func printVersion(stdout io.Writer) int {
	err := writeLine(stdout, version)
	if err != nil {
		return usageExitCode
	}

	return 0
}

func newFlagSet(stderr io.Writer, opts *options) *flag.FlagSet {
	flagSet := flag.NewFlagSet("coverlint", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flagSet.Float64Var(
		&opts.min,
		"min",
		coverlint.DefaultMinimum,
		"minimum coverage percentage for all packages",
	)
	flagSet.Var(&opts.overrides, "override", "package override GLOB=MIN; repeatable")
	flagSet.Var(&opts.excludes, "exclude", "package import-path glob to skip; repeatable")
	flagSet.DurationVar(&opts.timeout, "timeout", defaultTimeout, "maximum duration")
	flagSet.Var(&opts.testArgs, "test-arg", "additional go test argument; repeatable")
	flagSet.BoolVar(&opts.web, "web", false, "open the standard Go HTML coverage report")
	flagSet.BoolVar(&opts.showVersion, "version", false, "print version and exit")

	return flagSet
}

func runCoverage(opts options, args []string, stdout, stderr io.Writer) int {
	overrides, err := validateCoverageOptions(opts)
	if err != nil {
		return printUsageError(stderr, err)
	}

	runResult, err := coverlint.Check(context.Background(), coverlint.Config{
		Min:       opts.min,
		Overrides: overrides,
		Exclude:   []string(opts.excludes),
		Packages:  nil,
		Timeout:   opts.timeout.String(),
		TestArgs:  []string(opts.testArgs),
	}, args...)
	if err != nil {
		return printUsageError(stderr, err)
	}

	exitCode := reportCoverage(stdout, stderr, runResult)

	if exitCode != 0 {
		return exitCode
	}

	return openWebIfRequested(opts, runResult, stdout, stderr)
}

func validateCoverageOptions(opts options) ([]coverlint.Override, error) {
	overrides, err := parseOverrides(opts.overrides)
	if err != nil {
		return nil, err
	}

	err = coverlint.ValidateMinimum(opts.min)
	if err != nil {
		return nil, fmt.Errorf("validate minimum: %w", err)
	}

	if opts.timeout <= 0 {
		return nil, errNonPositiveValue
	}

	return overrides, nil
}

func printUsageError(stderr io.Writer, err error) int {
	writeErr := printError(stderr, err)
	if writeErr != nil {
		return usageExitCode
	}

	return usageExitCode
}

func reportCoverage(stdout, stderr io.Writer, runResult coverlint.Run) int {
	err := writeDiagnostics(stdout, runResult.Report.Results)
	if err != nil {
		return usageExitCode
	}

	err = writeSummary(stderr, runResult.Report)
	if err != nil {
		return usageExitCode
	}

	if runResult.Report.Failed > 0 {
		return 1
	}

	return 0
}

func writeDiagnostics(stdout io.Writer, results []coverlint.Result) error {
	for _, result := range results {
		if !result.Violation {
			continue
		}

		err := writeLine(stdout, domain.Diagnostic(result, coverlint.Name))
		if err != nil {
			return err
		}
	}

	return nil
}

func writeSummary(stderr io.Writer, report coverlint.Report) error {
	if report.Failed == 0 {
		return writeFormatted(
			stderr,
			"coverlint: passed (%d checked, %d skipped)\n",
			report.Checked,
			report.Skipped,
		)
	}

	return writeFormatted(
		stderr,
		"coverlint: failed with %d issue(s) (%d checked, %d skipped)\n",
		report.Failed,
		report.Checked,
		report.Skipped,
	)
}

func openWebIfRequested(opts options, runResult coverlint.Run, stdout, stderr io.Writer) int {
	if !opts.web {
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)

	defer cancel()

	err := runResult.OpenWeb(ctx, stdout, stderr)
	if err != nil {
		return printUsageError(stderr, err)
	}

	return 0
}

func printUsage(stderr io.Writer, flagSet *flag.FlagSet) error {
	lines := []string{
		"Usage: coverlint [flags] [package-pattern ...]",
		"",
		"Examples:",
		"  coverlint",
		"  coverlint -min 85 ./...",
		"  coverlint -min 75 -override '**/critical/**=95'",
		"",
	}

	for _, line := range lines {
		err := writeLine(stderr, line)
		if err != nil {
			return err
		}
	}

	flagSet.PrintDefaults()

	return nil
}

func writeLine(writer io.Writer, values ...any) error {
	_, err := fmt.Fprintln(writer, values...)
	if err != nil {
		return fmt.Errorf("write line: %w", err)
	}

	return nil
}

func writeFormatted(writer io.Writer, format string, values ...any) error {
	_, err := fmt.Fprintf(writer, format, values...)
	if err != nil {
		return fmt.Errorf("write formatted line: %w", err)
	}

	return nil
}

func printError(stderr io.Writer, err error) error {
	_, writeErr := fmt.Fprintf(stderr, "coverlint: %v\n", err)
	if writeErr != nil {
		return fmt.Errorf("write coverlint error: %w", writeErr)
	}

	return nil
}

func parseOverrides(values []string) ([]coverlint.Override, error) {
	overrides := make([]coverlint.Override, 0, len(values))

	for index, value := range values {
		override, err := parseOverride(index+1, value)
		if err != nil {
			return nil, err
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
}

func parseOverride(index int, value string) (coverlint.Override, error) {
	pattern, minimumText, ok := splitOverride(value)

	if !ok {
		return coverlint.Override{}, overrideFormatError{index: index, value: value}
	}

	minimum, err := strconv.ParseFloat(minimumText, 64)
	if err != nil {
		return coverlint.Override{}, fmt.Errorf(
			"override %d %q has invalid minimum %q: %w",
			index,
			value,
			minimumText,
			err,
		)
	}

	return coverlint.Override{Pattern: pattern, Min: minimum}, nil
}

func splitOverride(value string) (string, string, bool) {
	separator := strings.LastIndex(value, "=")

	if separator <= 0 || separator == len(value)-1 {
		return "", "", false
	}

	minimumText := strings.TrimSuffix(strings.TrimSpace(value[separator+1:]), "%")

	return value[:separator], minimumText, true
}

func writerFromOS(name string) io.Writer {
	switch name {
	case "stdout":
		return osStdout()
	default:
		return osStderr()
	}
}
