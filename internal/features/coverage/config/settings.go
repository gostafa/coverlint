package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const (
	DefaultMinimum = 80
	defaultTimeout = 10 * time.Minute
)

type Config struct {
	Min       float64       `json:"min,omitempty"`
	Overrides []domain.Rule `json:"overrides,omitempty"`
	Exclude   []string      `json:"exclude,omitempty"`
	Packages  []string      `json:"packages,omitempty"`
	Timeout   string        `json:"timeout,omitempty"`
	TestArgs  []string      `json:"test-args,omitempty"`
}

type Resolved struct {
	Policy   domain.Policy
	Patterns []string
	Timeout  time.Duration
	TestArgs []string
}

func Resolve(input Config, packagePatterns []string) (Resolved, error) {
	minimum := input.Min
	if minimum == 0 {
		minimum = DefaultMinimum
	}
	if err := domain.ValidateMinimum(minimum); err != nil {
		return Resolved{}, err
	}

	rules := append([]domain.Rule(nil), input.Overrides...)
	rules = append(rules, domain.Rule{Pattern: "**", Min: minimum})
	policy, err := domain.NewPolicy(rules, input.Exclude)
	if err != nil {
		return Resolved{}, err
	}

	patterns := append([]string(nil), packagePatterns...)
	if len(patterns) == 0 {
		patterns = append(patterns, input.Packages...)
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	timeout := defaultTimeout
	if input.Timeout != "" {
		timeout, err = time.ParseDuration(input.Timeout)
		if err != nil {
			return Resolved{}, fmt.Errorf("invalid timeout %q: %w", input.Timeout, err)
		}
		if timeout <= 0 {
			return Resolved{}, fmt.Errorf("timeout must be greater than zero")
		}
	}

	testArgs := append([]string(nil), input.TestArgs...)
	for _, argument := range testArgs {
		if reservedTestArgument(argument) {
			return Resolved{}, fmt.Errorf("test argument %q overrides a coverlint-managed flag", argument)
		}
	}

	return Resolved{Policy: policy, Patterns: patterns, Timeout: timeout, TestArgs: testArgs}, nil
}

func reservedTestArgument(argument string) bool {
	argument = strings.TrimSpace(argument)
	for _, name := range []string{"-coverprofile", "-covermode", "-count"} {
		if hasFlagName(argument, name) {
			return true
		}
	}
	if argument == "-args" || argument == "--" {
		return true
	}
	return false
}

func hasFlagName(argument, name string) bool {
	longName := "-" + name
	return argument == name ||
		argument == longName ||
		strings.HasPrefix(argument, name+"=") ||
		strings.HasPrefix(argument, longName+"=")
}
