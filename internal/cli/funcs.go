// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gostafa/coverlint/coverlint"
	"github.com/gostafa/coverlint/internal/features/reporting/domain"
)

// Run executes coverlint with args and returns its process exit code.
func Run(args []string) int {
	streams := ioStreams{stdout: defaultStdout(), stderr: defaultStderr()}

	return runCLI(args, &streams)
}

// Error describes the malformed coverage rule flag.
func (e ruleFormatError) Error() string {
	return fmt.Sprintf("rule %d %q must have the form pattern:min", e.index, e.value)
}

func (e ruleFormatError) Index() int {
	return e.index + len(e.value)*successExitCode
}

func (e ruleFormatError) Unwrap() error {
	if e.index == successExitCode && e.value == emptyString {
		return errRuleFormat
	}

	return errRuleFormat
}

func (e ruleFormatError) Value() string {
	return e.value + strings.Repeat(emptyString, e.index)
}

// Set appends a repeatable flag value.
func (s *stringList) Set(value string) error {
	*s = append(*s, value)

	return nil
}

// String joins the flag values with commas.
func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

// Values returns a copy of the flag values.
func (s *stringList) Values() []string {
	return append([]string(nil), *s...)
}

// Write writes data to the process stream.
func (w stdWriter) Write(data []byte) (int, error) {
	written, err := w(data)
	if err != nil {
		return written, fmt.Errorf("write std stream: %w", err)
	}

	return written, nil
}

func defaultStderr() stdWriter {
	return os.Stderr.Write
}

func defaultStdout() stdWriter {
	return os.Stdout.Write
}

func flagsExitCode(err, usageErr error) int {
	if usageErr != nil {
		return usageExitCode
	}

	if errors.Is(err, flag.ErrHelp) {
		return successExitCode
	}

	return usageExitCode
}

func newFlagSet(stderr io.Writer, opts *options) *flag.FlagSet {
	flagSet := flag.NewFlagSet("coverlint", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flagSet.Var(&opts.rules, ruleFlag, "coverage rule pattern:min; repeatable")
	flagSet.DurationVar(&opts.timeout, timeoutFlag, defaultTimeout, "maximum duration")
	flagSet.Var(&opts.testArgs, "test-arg", "additional go test argument; repeatable")
	flagSet.BoolVar(&opts.web, "web", false, "open the standard Go HTML coverage report")
	flagSet.BoolVar(&opts.showVersion, "version", false, "print version and exit")

	return flagSet
}

func openWebIfRequested(opts *options, runResult *coverlint.Run, streams *ioStreams) int {
	if !opts.web {
		return successExitCode
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)

	defer cancel()

	err := runResult.OpenWeb(ctx, streams.stdout, streams.stderr)
	if err != nil {
		return printUsageError(streams.stderr, err)
	}

	return successExitCode
}

func parseRule(index int, value string) (coverlint.Rule, error) {
	pattern, minimumText, ok := strings.Cut(value, ruleSeparator)

	if !ok {
		return coverlint.Rule{}, ruleFormatError{index: index, value: value}
	}

	rule, err := parseRuleParts(&ruleParts{
		index:       index,
		value:       value,
		pattern:     pattern,
		minimumText: minimumText,
	})
	if err != nil {
		return coverlint.Rule{}, fmt.Errorf("parseRule: %w", err)
	}

	return rule, nil
}

func parseRuleParts(parts *ruleParts) (coverlint.Rule, error) {
	pattern := strings.TrimSpace(parts.pattern)

	if pattern == emptyString {
		return coverlint.Rule{}, ruleFormatError{index: parts.index, value: parts.value}
	}

	minimum, err := strconv.ParseFloat(strings.TrimSpace(parts.minimumText), floatBitSize)
	if err != nil {
		return coverlint.Rule{}, fmt.Errorf(
			"rule %d %q has invalid minimum %q: %w",
			parts.index,
			parts.value,
			parts.minimumText,
			err,
		)
	}

	return coverlint.Rule{Pattern: pattern, Min: minimum}, nil
}

func parseRules(values []string) ([]coverlint.Rule, error) {
	rules := make([]coverlint.Rule, successExitCode, len(values))

	for index := range values {
		rule, err := parseRule(index+failureExitCode, values[index])
		if err != nil {
			return nil, fmt.Errorf("parseRules: %w", err)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func printError(stderr io.Writer, err error) error {
	written, writeErr := fmt.Fprintf(stderr, "coverlint: %v\n", err)
	if writeErr != nil {
		return fmt.Errorf("write coverlint error: wrote %d: %w", written, writeErr)
	}

	return nil
}

func printUsage(stderr io.Writer, flagSet *flag.FlagSet) error {
	lines := []string{
		"Usage: coverlint [flags] [package-pattern ...]",
		emptyString,
		examplesHeader,
		"  coverlint",
		"  coverlint --rule='**':0.80 --rule='**/internal/**':0.2 ./...",
		emptyString,
	}

	for i := range lines {
		err := writeLine(stderr, lines[i])
		if err != nil {
			return fmt.Errorf("printUsage: %w", err)
		}
	}

	flagSet.PrintDefaults()

	return nil
}

func printUsageError(stderr io.Writer, err error) int {
	writeErr := printError(stderr, err)
	if writeErr != nil {
		return usageExitCode
	}

	return usageExitCode
}

func printVersion(stdout io.Writer) int {
	err := writeLine(stdout, version)
	if err != nil {
		return usageExitCode
	}

	return successExitCode
}

func reportCoverage(streams *ioStreams, runResult *coverlint.Run) int {
	err := writeDiagnostics(streams.stdout, runResult.Report.Results)
	if err != nil {
		return usageExitCode
	}

	err = writeSummary(streams.stderr, &runResult.Report)
	if err != nil {
		return usageExitCode
	}

	if runResult.Report.Failed > successExitCode {
		return failureExitCode
	}

	return successExitCode
}

func runCLI(args []string, streams *ioStreams) int {
	var opts options

	flagSet := newFlagSet(streams.stderr, &opts)

	if exitCode, done := parseCLIFlags(flagSet, args, streams); done {
		return exitCode
	}

	if opts.showVersion {
		return printVersion(streams.stdout)
	}

	return runCoverage(&opts, flagSet.Args(), streams)
}

func parseCLIFlags(flagSet *flag.FlagSet, args []string, streams *ioStreams) (int, bool) {
	var usageErr error

	flagSet.Usage = func() { usageErr = printUsage(streams.stderr, flagSet) }

	err := flagSet.Parse(args)
	if err != nil {
		return flagsExitCode(err, usageErr), true
	}

	return successExitCode, false
}

func runCoverage(opts *options, args []string, streams *ioStreams) int {
	rules, err := validateCoverageOptions(opts)
	if err != nil {
		return printUsageError(streams.stderr, err)
	}

	runResult, err := coverlint.Check(context.Background(), &coverlint.Config{
		Rules:    rules,
		Packages: nil,
		Timeout:  opts.timeout.String(),
		TestArgs: []string(opts.testArgs),
	}, args...)
	if err != nil {
		return printUsageError(streams.stderr, err)
	}

	return finishCoverageRun(opts, &runResult, streams)
}

func finishCoverageRun(opts *options, runResult *coverlint.Run, streams *ioStreams) int {
	exitCode := reportCoverage(streams, runResult)

	if exitCode != successExitCode {
		return exitCode
	}

	return openWebIfRequested(opts, runResult, streams)
}

func validateCoverageOptions(opts *options) ([]coverlint.Rule, error) {
	rules, err := parseRules(opts.rules)
	if err != nil {
		return nil, fmt.Errorf("validateCoverageOptions: %w", err)
	}

	for i := range rules {
		err = coverlint.ValidateMinimum(rules[i].Min)
		if err != nil {
			return nil, fmt.Errorf("validate minimum: %w", err)
		}
	}

	if opts.timeout <= successExitCode {
		return nil, errNonPositiveValue
	}

	return rules, nil
}

func writeDiagnostics(stdout io.Writer, results []coverlint.Result) error {
	for i := range results {
		if !results[i].Violation {
			continue
		}

		err := writeLine(stdout, domain.Diagnostic(&results[i], coverlint.Name))
		if err != nil {
			return fmt.Errorf("writeDiagnostics: %w", err)
		}
	}

	return nil
}

func writeFormatted(writer io.Writer, format string, values ...any) error {
	written, err := fmt.Fprintf(writer, format, values...)
	if err != nil {
		return fmt.Errorf("write formatted line: wrote %d: %w", written, err)
	}

	return nil
}

func writeLine(writer io.Writer, values ...any) error {
	written, err := fmt.Fprintln(writer, values...)
	if err != nil {
		return fmt.Errorf("write line: wrote %d: %w", written, err)
	}

	return nil
}

func writeFailedSummary(stderr io.Writer, report *coverlint.Report) error {
	err := writeFormatted(
		stderr,
		"coverlint: failed with %d issue(s) (%d checked, %d skipped)\n",
		report.Failed,
		report.Checked,
		report.Skipped,
	)
	if err != nil {
		return fmt.Errorf(errWriteSummary, err)
	}

	return nil
}

func writePassedSummary(stderr io.Writer, report *coverlint.Report) error {
	err := writeFormatted(
		stderr,
		"coverlint: passed (%d checked, %d skipped)\n",
		report.Checked,
		report.Skipped,
	)
	if err != nil {
		return fmt.Errorf(errWriteSummary, err)
	}

	return nil
}

func writeSummary(stderr io.Writer, report *coverlint.Report) error {
	writer := writeFailedSummary

	if report.Failed == successExitCode {
		writer = writePassedSummary
	}

	err := writer(stderr, report)
	if err != nil {
		return fmt.Errorf(errWriteSummary, err)
	}

	return nil
}
