package config_test

import (
	"math"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/config"
	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

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
