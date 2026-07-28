package config_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const configuredPattern = "./configured"

func TestResolveRejectsNonFiniteMinimums(t *testing.T) {
	t.Parallel()

	for _, minimum := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1, 100.1} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			input := testConfig()
			input.Min = minimum

			_, err := config.Resolve(input, nil)
			if err == nil {
				t.Fatalf("Resolve minimum %v succeeded, want error", minimum)
			}
		})
	}
}

func TestResolveUsesDefaultMinimumForZero(t *testing.T) {
	t.Parallel()

	input := testConfig()

	_, err := config.Resolve(input, nil)
	if err != nil {
		t.Fatalf("Resolve default minimum: %v", err)
	}
}

func TestResolveRejectsNonFiniteOverrideMinimums(t *testing.T) {
	t.Parallel()

	input := testConfig()
	input.Overrides = []domain.Rule{{Pattern: "**/critical/**", Min: math.NaN()}}

	_, err := config.Resolve(input, nil)
	if err == nil {
		t.Fatal("Resolve accepted NaN override minimum, want error")
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

			_, err := config.Resolve(input, nil)
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

	resolved, err := config.Resolve(input, []string{"./requested"})
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

	resolved, err := config.Resolve(input, nil)
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

			_, err := config.Resolve(input, nil)
			if err == nil {
				t.Fatalf("Resolve accepted timeout %q, want error", timeout)
			}
		})
	}
}

func TestResolveWrapsPolicyErrors(t *testing.T) {
	t.Parallel()

	input := testConfig()
	input.Overrides = []domain.Rule{{Pattern: "[", Min: 80}}

	_, err := config.Resolve(input, nil)
	if err == nil {
		t.Fatal("Resolve accepted invalid glob, want error")
	}
}

func testConfig() config.Config {
	return config.Config{
		Min:       0,
		Overrides: nil,
		Exclude:   nil,
		Packages:  nil,
		Timeout:   "",
		TestArgs:  nil,
	}
}
