// Package config resolves coverage feature settings into executable requests.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const (
	// DefaultMinimum is the default required package coverage percentage.
	DefaultMinimum = 80
	defaultTimeout = 10 * time.Minute
)

var (
	errNonPositiveTimeout = errors.New("timeout must be greater than zero")
	errManagedTestFlag    = errors.New("test argument overrides a coverlint-managed flag")
)

// Config contains user-provided coverage settings.
type Config struct {
	Min       float64       `json:"min,omitempty"`
	Overrides []domain.Rule `json:"overrides,omitempty"`
	Exclude   []string      `json:"exclude,omitempty"`
	Packages  []string      `json:"packages,omitempty"`
	Timeout   string        `json:"timeout,omitempty"`
	TestArgs  []string      `json:"testArgs,omitempty"  mapstructure:"test-args"`
}

// Resolved contains validated settings ready for coverage execution.
type Resolved struct {
	Policy   domain.Policy
	Patterns []string
	Timeout  time.Duration
	TestArgs []string
}

// Resolve validates and expands user-provided coverage settings.
func Resolve(input Config, packagePatterns []string) (Resolved, error) {
	minimum, err := resolveMinimum(input.Min)
	if err != nil {
		return Resolved{}, err
	}

	rules := append([]domain.Rule(nil), input.Overrides...)
	rules = append(rules, domain.Rule{Pattern: "**", Min: minimum})

	policy, err := domain.NewPolicy(rules, input.Exclude)
	if err != nil {
		return Resolved{}, fmt.Errorf("build coverage policy: %w", err)
	}

	timeout, err := resolveTimeout(input.Timeout)
	if err != nil {
		return Resolved{}, err
	}

	testArgs := append([]string(nil), input.TestArgs...)
	for _, argument := range testArgs {
		if reservedTestArgument(argument) {
			return Resolved{}, fmt.Errorf(
				"%w: %q",
				errManagedTestFlag,
				argument,
			)
		}
	}

	return Resolved{
		Policy:   policy,
		Patterns: resolvePatterns(input.Packages, packagePatterns),
		Timeout:  timeout,
		TestArgs: testArgs,
	}, nil
}

func resolveMinimum(value float64) (float64, error) {
	if value == 0 {
		value = DefaultMinimum
	}

	err := domain.ValidateMinimum(value)
	if err != nil {
		return 0, fmt.Errorf("validate minimum: %w", err)
	}

	return value, nil
}

func resolvePatterns(configured []string, requested []string) []string {
	patterns := append([]string(nil), requested...)
	if len(patterns) == 0 {
		patterns = append(patterns, configured...)
	}

	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	return patterns
}

func resolveTimeout(value string) (time.Duration, error) {
	if value == "" {
		return defaultTimeout, nil
	}

	timeout, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", value, err)
	}

	if timeout <= 0 {
		return 0, errNonPositiveTimeout
	}

	return timeout, nil
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
