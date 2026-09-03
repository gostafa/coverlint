// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	errUnknownConfigField = errors.New("unknown coverage config field")
	errAmbiguousTestArgs  = errors.New("ambiguous coverage config")
)

// Config contains user-provided coverage settings.
type Config struct {
	Min       float64       `json:"min,omitempty"`
	Overrides []domain.Rule `json:"overrides,omitempty"`
	Exclude   []string      `json:"exclude,omitempty"`
	Packages  []string      `json:"packages,omitempty"`
	Timeout   string        `json:"timeout,omitempty"`
	TestArgs  []string      `json:"testArgs,omitempty"`
}

// UnmarshalJSON accepts the documented legacy test-args key and the camelCase key.
func (c *Config) UnmarshalJSON(data []byte) error {
	raw, err := decodeRawConfig(data)
	if err != nil {
		return err
	}

	err = validateRawConfig(raw)
	if err != nil {
		return err
	}

	decoded, err := decodeConfigAlias(data)
	if err != nil {
		return err
	}

	testArgs, err := decodeTestArgs(raw)
	if err != nil {
		return err
	}

	decoded.TestArgs = testArgs
	*c = decoded

	return nil
}

func decodeRawConfig(data []byte) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("decode coverage config: %w", err)
	}

	return raw, nil
}

func validateRawConfig(raw map[string]json.RawMessage) error {
	err := validateConfigKeys(raw)
	if err != nil {
		return err
	}

	return validateTestArgsKeys(raw)
}

func validateConfigKeys(raw map[string]json.RawMessage) error {
	for key := range raw {
		err := validateConfigKey(key)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateConfigKey(key string) error {
	if slices.Contains(configKeys(), key) {
		return nil
	}

	return fmt.Errorf("%w: %q", errUnknownConfigField, key)
}

func configKeys() []string {
	return []string{
		"min",
		"overrides",
		"exclude",
		"packages",
		"timeout",
		testArgsKey(),
		legacyTestArgsKey(),
	}
}

func validateTestArgsKeys(raw map[string]json.RawMessage) error {
	if _, ok := raw[testArgsKey()]; ok {
		if _, legacy := raw[legacyTestArgsKey()]; legacy {
			return fmt.Errorf(
				"%w: use only one of %q or %q",
				errAmbiguousTestArgs,
				testArgsKey(),
				legacyTestArgsKey(),
			)
		}
	}

	return nil
}

func decodeConfigAlias(data []byte) (Config, error) {
	type configAlias Config

	var decoded configAlias

	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return Config{}, fmt.Errorf("decode coverage config fields: %w", err)
	}

	return Config(decoded), nil
}

func decodeTestArgs(raw map[string]json.RawMessage) ([]string, error) {
	value, ok := raw[testArgsKey()]

	if !ok {
		value = raw[legacyTestArgsKey()]
	}

	if len(value) == 0 {
		return nil, nil
	}

	var testArgs []string

	err := json.Unmarshal(value, &testArgs)
	if err != nil {
		return nil, fmt.Errorf("decode test args: %w", err)
	}

	return testArgs, nil
}

func testArgsKey() string {
	return "testArgs"
}

func legacyTestArgsKey() string {
	return "test-args"
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

	testArgs, err := resolveTestArgs(input.TestArgs)
	if err != nil {
		return Resolved{}, err
	}

	return Resolved{
		Policy:   policy,
		Patterns: resolvePatterns(input.Packages, packagePatterns),
		Timeout:  timeout,
		TestArgs: testArgs,
	}, nil
}

func resolveTestArgs(input []string) ([]string, error) {
	testArgs := append([]string(nil), input...)

	for _, argument := range testArgs {
		err := validateTestArgument(argument)
		if err != nil {
			return nil, err
		}
	}

	return testArgs, nil
}

func validateTestArgument(argument string) error {
	if !reservedTestArgument(argument) {
		return nil
	}

	return fmt.Errorf("%w: %q", errManagedTestFlag, argument)
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
		patterns = defaultPatterns()
	}

	return patterns
}

func defaultPatterns() []string {
	return []string{"./..."}
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
