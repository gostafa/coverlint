// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package config_test

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const configuredPattern = "./configured"

func TestConfigUnmarshalAcceptsDocumentedTestArgsKey(t *testing.T) {
	t.Parallel()

	var got config.Config

	err := config.Unmarshal([]byte(`{"test-args":["-race","-tags=integration"]}`), &got)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	want := []string{"-race", "-tags=integration"}

	if !reflect.DeepEqual(got.TestArgs, want) {
		t.Fatalf("TestArgs = %#v, want %#v", got.TestArgs, want)
	}
}

func TestConfigUnmarshalAcceptsCamelCaseTestArgsKey(t *testing.T) {
	t.Parallel()

	var got config.Config

	err := config.Unmarshal([]byte(`{"testArgs":["-race"]}`), &got)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	want := []string{"-race"}

	if !reflect.DeepEqual(got.TestArgs, want) {
		t.Fatalf("TestArgs = %#v, want %#v", got.TestArgs, want)
	}
}

func TestConfigUnmarshalAcceptsSnakeCaseTestArgsKey(t *testing.T) {
	t.Parallel()

	var got config.Config

	err := config.Unmarshal([]byte(`{"test_args":["-race"]}`), &got)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	want := []string{"-race"}

	if !reflect.DeepEqual(got.TestArgs, want) {
		t.Fatalf("TestArgs = %#v, want %#v", got.TestArgs, want)
	}
}

func TestConfigUnmarshalRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	var got config.Config

	err := config.Unmarshal([]byte(`{"minimum":85}`), &got)

	if err == nil || !strings.Contains(err.Error(), `unknown coverage config field: "minimum"`) {
		t.Fatalf("error = %v, want unknown field error", err)
	}
}

func TestConfigUnmarshalRejectsAmbiguousTestArgsKeys(t *testing.T) {
	t.Parallel()

	var got config.Config

	err := config.Unmarshal([]byte(`{"testArgs":["-race"],"test-args":["-run","TestUnit"]}`), &got)

	if err == nil || !strings.Contains(err.Error(), "ambiguous coverage config") {
		t.Fatalf("error = %v, want ambiguous test args error", err)
	}
}

func TestResolveRejectsNonFiniteRuleMinimums(t *testing.T) {
	t.Parallel()

	for _, minimum := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1, 1.1} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			input := testConfig()

			input.Rules = []domain.Rule{{Pattern: "**", Min: minimum}}

			_, err := config.Resolve(&input, nil)
			if err == nil {
				t.Fatalf("Resolve minimum %v succeeded, want error", minimum)
			}
		})
	}
}

func TestResolveUsesDefaultRulesWhenEmpty(t *testing.T) {
	t.Parallel()

	input := testConfig()

	_, err := config.Resolve(&input, nil)
	if err != nil {
		t.Fatalf("Resolve default rules: %v", err)
	}
}

func TestResolveAllowsZeroRuleMinimum(t *testing.T) {
	t.Parallel()

	input := testConfig()

	input.Rules = []domain.Rule{{Pattern: "**/*_test", Min: 0}}

	_, err := config.Resolve(&input, nil)
	if err != nil {
		t.Fatalf("Resolve zero minimum: %v", err)
	}
}

func TestConfigUnmarshalRejectsLegacyMinAndOverrides(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"min":0.80}`, `{"overrides":[]}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			var got config.Config

			err := config.Unmarshal([]byte(body), &got)

			if err == nil || !strings.Contains(err.Error(), "unknown coverage config field") {
				t.Fatalf("error = %v, want unknown field error", err)
			}
		})
	}
}

func TestConfigUnmarshalAcceptsRules(t *testing.T) {
	t.Parallel()

	var got config.Config

	err := config.Unmarshal([]byte(`{"rules":[{"pattern":"**","min":0.80}]}`), &got)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	want := []domain.Rule{{Pattern: "**", Min: 0.80}}

	if !reflect.DeepEqual(got.Rules, want) {
		t.Fatalf("Rules = %#v, want %#v", got.Rules, want)
	}
}

func TestResolveRejectsReservedTestArguments(t *testing.T) {
	t.Parallel()

	for _, argument := range []string{
		"-coverprofile", "-coverprofile=out",
		"--coverprofile", "--coverprofile=out",
		"-covermode", "--covermode=atomic",
		"-count", "--count=1",
		"-args", "--",
	} {
		t.Run(argument, func(t *testing.T) {
			t.Parallel()

			input := testConfig()

			input.TestArgs = []string{argument}

			_, err := config.Resolve(&input, nil)
			if err == nil {
				t.Fatalf("Resolve accepted reserved test argument %q, want error", argument)
			}
		})
	}
}

func TestResolveUsesRequestedPatternsBeforeConfiguredPatterns(t *testing.T) {
	t.Parallel()

	input := testConfig()

	input.Packages = []string{configuredPattern}
	input.TestArgs = []string{"-run", "TestUnit"}

	resolved, err := config.Resolve(&input, []string{"./requested"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if !reflect.DeepEqual(resolved.Patterns, []string{"./requested"}) {
		t.Fatalf("Patterns = %#v, want requested pattern", resolved.Patterns)
	}

	if !reflect.DeepEqual(resolved.TestArgs, []string{"-run", "TestUnit"}) {
		t.Fatalf("TestArgs = %#v", resolved.TestArgs)
	}
}

func TestResolveUsesConfiguredPatternsAndTimeout(t *testing.T) {
	t.Parallel()

	input := testConfig()

	input.Packages = []string{configuredPattern}
	input.Timeout = "5s"

	resolved, err := config.Resolve(&input, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if !reflect.DeepEqual(resolved.Patterns, []string{configuredPattern}) {
		t.Fatalf("Patterns = %#v, want configured pattern", resolved.Patterns)
	}

	if resolved.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %s, want 5s", resolved.Timeout)
	}
}

func TestResolveRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()

	for _, timeout := range []string{"nope", "0s", "-1s"} {
		t.Run(timeout, func(t *testing.T) {
			t.Parallel()

			input := testConfig()

			input.Timeout = timeout

			_, err := config.Resolve(&input, nil)
			if err == nil {
				t.Fatalf("Resolve accepted timeout %q, want error", timeout)
			}
		})
	}
}

func TestResolveWrapsPolicyErrors(t *testing.T) {
	t.Parallel()

	input := testConfig()

	input.Rules = []domain.Rule{{Pattern: "[", Min: 0.80}}

	_, err := config.Resolve(&input, nil)
	if err == nil {
		t.Fatal("Resolve accepted invalid glob, want error")
	}
}

func testConfig() config.Config {
	return config.Config{
		Rules:    nil,
		Exclude:  nil,
		Packages: nil,
		Timeout:  "",
		TestArgs: nil,
	}
}
