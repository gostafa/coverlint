// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func (fn decoderFunc) decode(data []byte) error {
	err := fn(data)
	if err != nil {
		return fmt.Errorf("decode coverage config: %w", err)
	}

	return nil
}

// Unmarshal accepts test_args plus legacy testArgs / test-args aliases.
func Unmarshal(data []byte, cfg *Config) error {
	remapped, err := remapTestArgsKeys(data)
	if err != nil {
		return fmt.Errorf(errUnmarshalCoverageConfig, err)
	}

	err = finishUnmarshal(cfg, remapped)
	if err != nil {
		return fmt.Errorf(errUnmarshalCoverageConfig, err)
	}

	return nil
}

func applyDecodedConfig(cfg *Config, data []byte) error {
	decoded, err := decodeConfigAlias(data)
	if err != nil {
		return fmt.Errorf("apply coverage config: %w", err)
	}

	*cfg = decoded

	return nil
}

// Resolve validates and expands user-provided coverage settings.
func Resolve(input *Config, packagePatterns []string) (Resolved, error) {
	if input == nil {
		input = &Config{}
	}

	policy, err := domain.NewPolicy(resolveRules(input.Rules))
	if err != nil {
		return Resolved{}, fmt.Errorf("build coverage policy: %w", err)
	}

	resolved, err := resolveWithPolicy(input, packagePatterns, &policy)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve coverage config: %w", err)
	}

	return resolved, nil
}

func resolveWithPolicy(
	input *Config,
	packagePatterns []string,
	policy *domain.Policy,
) (Resolved, error) {
	timeout, err := resolveTimeout(input.Timeout)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve coverage timeout: %w", err)
	}

	testArgs, err := resolveTestArgs(input.TestArgs)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve coverage test args: %w", err)
	}

	return Resolved{
		Policy:   *policy,
		Patterns: resolvePatterns(input.Packages, packagePatterns),
		Timeout:  timeout,
		TestArgs: testArgs,
	}, nil
}

func configKeys() []string {
	return []string{
		"rules",
		"packages",
		timeoutKey,
		testArgsKey,
	}
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

func decodeRawConfig(data []byte) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf(errDecodeCoverageConfig, err)
	}

	return raw, nil
}

func defaultPatterns() []string {
	return []string{defaultPattern}
}

func hasFlagName(argument, name string) bool {
	longName := flagPrefix + name

	return argument == name ||
		argument == longName ||
		strings.HasPrefix(argument, name+equalsSign) ||
		strings.HasPrefix(argument, longName+equalsSign)
}

func canonicalizeTestArgsKey(raw map[string]json.RawMessage, present []string) {
	if len(present) != one {
		return
	}

	if present[zero] == testArgsKey {
		return
	}

	raw[testArgsKey] = raw[present[zero]]
	delete(raw, present[zero])
}

func finishUnmarshal(cfg *Config, remapped []byte) error {
	raw, err := decodeRawConfig(remapped)
	if err != nil {
		return fmt.Errorf(errUnmarshalCoverageConfig, err)
	}

	err = validateConfigKeys(raw)
	if err != nil {
		return fmt.Errorf(errUnmarshalCoverageConfig, err)
	}

	err = applyDecodedConfig(cfg, remapped)
	if err != nil {
		return fmt.Errorf(errUnmarshalCoverageConfig, err)
	}

	return nil
}

func marshalRawConfigWith(
	raw map[string]json.RawMessage,
	marshal func(any) ([]byte, error),
) ([]byte, error) {
	encoded, err := marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode remapped coverage config: %w", err)
	}

	return encoded, nil
}

func presentTestArgsKeys(raw map[string]json.RawMessage) []string {
	keys := []string{testArgsKey, testArgsCamel, testArgsLegacy}
	present := make([]string, zero, testArgsAliasCount)

	for index := range keys {
		key := keys[index]

		if _, ok := raw[key]; ok {
			present = append(present, key)
		}
	}

	return present
}

func remapTestArgsInRaw(raw map[string]json.RawMessage) error {
	present := presentTestArgsKeys(raw)

	err := validatePresentTestArgs(present)
	if err != nil {
		return fmt.Errorf(errRemapTestArgsKeys, err)
	}

	canonicalizeTestArgsKey(raw, present)

	return nil
}

func remapTestArgsKeys(data []byte) ([]byte, error) {
	encoded, err := remapTestArgsKeysWith(data, json.Marshal)
	if err != nil {
		return nil, fmt.Errorf(errRemapTestArgsKeys, err)
	}

	return encoded, nil
}

func remapTestArgsKeysWith(
	data []byte,
	marshal func(any) ([]byte, error),
) ([]byte, error) {
	raw, err := decodeRawConfig(data)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalCoverageConfig, err)
	}

	err = remapTestArgsInRaw(raw)
	if err != nil {
		return nil, fmt.Errorf(errRemapTestArgsKeys, err)
	}

	encoded, err := marshalRawConfigWith(raw, marshal)
	if err != nil {
		return nil, fmt.Errorf(errRemapTestArgsKeys, err)
	}

	return encoded, nil
}

func validatePresentTestArgs(present []string) error {
	if len(present) <= one {
		return nil
	}

	return fmt.Errorf(
		"%w: use only one of %q, %q, or %q",
		errAmbiguousTestArgs,
		testArgsKey,
		testArgsCamel,
		testArgsLegacy,
	)
}

func reservedTestArgument(argument string) bool {
	argument = strings.TrimSpace(argument)

	names := []string{"-coverprofile", "-covermode", "-count"}

	for i := range names {
		if hasFlagName(argument, names[i]) {
			return true
		}
	}

	return argument == "-args" || argument == doubleDash
}

func resolvePatterns(configured, requested []string) []string {
	patterns := append([]string(nil), requested...)

	if len(patterns) == zero {
		patterns = append(patterns, configured...)
	}

	if len(patterns) == zero {
		patterns = defaultPatterns()
	}

	return patterns
}

func resolveRules(rules []domain.Rule) []domain.Rule {
	if len(rules) == zero {
		return []domain.Rule{{Pattern: doubleStar, Min: DefaultMinimum}}
	}

	return rules
}

func resolveTestArgs(input []string) ([]string, error) {
	testArgs := append([]string(nil), input...)

	for i := range testArgs {
		err := validateTestArgument(testArgs[i])
		if err != nil {
			return nil, fmt.Errorf("resolve test args: %w", err)
		}
	}

	return testArgs, nil
}

func resolveTimeout(value string) (time.Duration, error) {
	if value == emptyString {
		return defaultTimeout, nil
	}

	timeout, err := time.ParseDuration(value)
	if err != nil {
		return zero, fmt.Errorf("invalid timeout %q: %w", value, err)
	}

	if timeout <= zero {
		return zero, errNonPositiveTimeout
	}

	return timeout, nil
}

func validateConfigKey(key string) error {
	if slices.Contains(configKeys(), key) {
		return nil
	}

	return fmt.Errorf(wrappedQuotedErr, errUnknownConfigField, key)
}

func validateConfigKeys(raw map[string]json.RawMessage) error {
	for key := range raw {
		err := validateConfigKey(key)
		if err != nil {
			return fmt.Errorf("validate coverage config keys: %w", err)
		}
	}

	return nil
}

func validateTestArgument(argument string) error {
	if !reservedTestArgument(argument) {
		return nil
	}

	return fmt.Errorf(wrappedQuotedErr, errManagedTestFlag, argument)
}
