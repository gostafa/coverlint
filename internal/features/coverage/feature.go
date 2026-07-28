package coverage

import (
	"context"
	"fmt"
	"io"

	"github.com/gostafa/coverlint/internal/features/coverage/adapters/gotool"
	"github.com/gostafa/coverlint/internal/features/coverage/adapters/text"
	"github.com/gostafa/coverlint/internal/features/coverage/application"
	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
)

const Name = "coverlint"
const DefaultMinimum = config.DefaultMinimum

type Override = domain.Rule
type Config = config.Config
type Result = domain.Result
type Report = domain.Report

type Run struct {
	Report  Report
	profile []byte
	html    ports.HTMLReporter
}

func Check(ctx context.Context, input Config, packagePatterns ...string) (Run, error) {
	resolved, err := config.Resolve(input, packagePatterns)
	if err != nil {
		return Run{}, err
	}

	toolchain := gotool.New()
	outcome, err := application.NewChecker(toolchain, toolchain).Check(ctx, application.Request{
		Policy:   resolved.Policy,
		Patterns: resolved.Patterns,
		Timeout:  resolved.Timeout,
		TestArgs: resolved.TestArgs,
	})
	if err != nil {
		return Run{}, err
	}
	return Run{Report: outcome.Report, profile: outcome.Profile, html: toolchain}, nil
}

func ValidateMinimum(value float64) error {
	return domain.ValidateMinimum(value)
}

func (r Run) OpenWeb(ctx context.Context, stdout, stderr io.Writer) error {
	if r.html == nil {
		return fmt.Errorf("HTML coverage adapter is not configured")
	}
	return r.html.Open(ctx, r.profile, stdout, stderr)
}

func Diagnostic(result Result) string {
	return text.Diagnostic(result, Name)
}
