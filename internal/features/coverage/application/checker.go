package application

import (
	"context"
	"fmt"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
	"github.com/gostafa/coverlint/internal/features/coverage/ports"
)

type Request struct {
	Policy   domain.Policy
	Patterns []string
	Timeout  time.Duration
	TestArgs []string
}

type Outcome struct {
	Report  domain.Report
	Profile []byte
}

type Checker struct {
	coverage ports.CoverageRunner
	packages ports.PackageCatalog
}

func NewChecker(coverage ports.CoverageRunner, packages ports.PackageCatalog) *Checker {
	return &Checker{coverage: coverage, packages: packages}
}

func (c *Checker) Check(parent context.Context, request Request) (Outcome, error) {
	if c == nil || c.coverage == nil || c.packages == nil {
		return Outcome{}, fmt.Errorf("coverage checker is not configured")
	}

	ctx, cancel := context.WithTimeout(parent, request.Timeout)
	defer cancel()

	coverage, err := c.coverage.Collect(ctx, ports.CoverageRequest{Patterns: request.Patterns, TestArgs: request.TestArgs})
	if err != nil {
		if ctx.Err() != nil {
			return Outcome{}, fmt.Errorf("coverage check exceeded timeout %s: %w", request.Timeout, ctx.Err())
		}
		return Outcome{}, err
	}

	packages, err := c.packages.List(ctx, ports.PackageRequest{Patterns: request.Patterns, TestArgs: request.TestArgs})
	if err != nil {
		if ctx.Err() != nil {
			return Outcome{}, fmt.Errorf("coverage check exceeded timeout %s: %w", request.Timeout, ctx.Err())
		}
		return Outcome{}, err
	}

	return Outcome{Report: request.Policy.Evaluate(packages, coverage.Blocks), Profile: coverage.Profile}, nil
}
