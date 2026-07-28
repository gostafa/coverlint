package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	coveragefeature "github.com/gostafa/coverlint/internal/features/coverage"
)

const version = "0.5.0"

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type options struct {
	min         float64
	overrides   stringList
	excludes    stringList
	timeout     time.Duration
	testArgs    stringList
	web         bool
	showVersion bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var opts options
	fs := flag.NewFlagSet("coverlint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Float64Var(&opts.min, "min", coveragefeature.DefaultMinimum, "minimum coverage percentage for all packages")
	fs.Var(&opts.overrides, "override", "package override GLOB=MIN; repeatable")
	fs.Var(&opts.excludes, "exclude", "package import-path glob to skip; repeatable")
	fs.DurationVar(&opts.timeout, "timeout", 10*time.Minute, "maximum duration")
	fs.Var(&opts.testArgs, "test-arg", "additional go test argument; repeatable")
	fs.BoolVar(&opts.web, "web", false, "open the standard Go HTML coverage report")
	fs.BoolVar(&opts.showVersion, "version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: coverlint [flags] [package-pattern ...]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Examples:")
		fmt.Fprintln(stderr, "  coverlint")
		fmt.Fprintln(stderr, "  coverlint -min 85 ./...")
		fmt.Fprintln(stderr, "  coverlint -min 75 -override '**/critical/**=95'")
		fmt.Fprintln(stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if opts.showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	overrides, err := parseOverrides(opts.overrides)
	if err != nil {
		fmt.Fprintf(stderr, "coverlint: %v\n", err)
		return 2
	}
	if err := coveragefeature.ValidateMinimum(opts.min); err != nil {
		fmt.Fprintf(stderr, "coverlint: %v\n", err)
		return 2
	}
	if opts.timeout <= 0 {
		fmt.Fprintln(stderr, "coverlint: timeout must be greater than zero")
		return 2
	}

	runResult, err := coveragefeature.Check(context.Background(), coveragefeature.Config{
		Min:       opts.min,
		Overrides: overrides,
		Exclude:   []string(opts.excludes),
		Timeout:   opts.timeout.String(),
		TestArgs:  []string(opts.testArgs),
	}, fs.Args()...)
	if err != nil {
		fmt.Fprintf(stderr, "coverlint: %v\n", err)
		return 2
	}

	for _, result := range runResult.Report.Results {
		if result.Violation {
			fmt.Fprintln(stdout, coveragefeature.Diagnostic(result))
		}
	}

	if runResult.Report.Failed == 0 {
		fmt.Fprintf(stderr, "coverlint: passed (%d checked, %d skipped)\n", runResult.Report.Checked, runResult.Report.Skipped)
	} else {
		fmt.Fprintf(stderr, "coverlint: failed with %d issue(s) (%d checked, %d skipped)\n", runResult.Report.Failed, runResult.Report.Checked, runResult.Report.Skipped)
	}

	if opts.web {
		ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
		defer cancel()
		if err := runResult.OpenWeb(ctx, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "coverlint: %v\n", err)
			return 2
		}
	}

	if runResult.Report.Failed > 0 {
		return 1
	}
	return 0
}

func parseOverrides(values []string) ([]coveragefeature.Override, error) {
	overrides := make([]coveragefeature.Override, 0, len(values))
	for i, value := range values {
		separator := strings.LastIndex(value, "=")
		if separator <= 0 || separator == len(value)-1 {
			return nil, fmt.Errorf("override %d %q must have the form GLOB=MIN", i+1, value)
		}
		pattern := value[:separator]
		minimumText := strings.TrimSuffix(strings.TrimSpace(value[separator+1:]), "%")
		minimum, err := strconv.ParseFloat(minimumText, 64)
		if err != nil {
			return nil, fmt.Errorf("override %d %q has invalid minimum %q: %w", i+1, value, minimumText, err)
		}
		overrides = append(overrides, coveragefeature.Override{Pattern: pattern, Min: minimum})
	}
	return overrides, nil
}
