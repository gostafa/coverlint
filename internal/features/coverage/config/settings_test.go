package config

import (
	"math"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

func TestResolveRejectsNonFiniteMinimums(t *testing.T) {
	t.Parallel()

	for _, minimum := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1, 100.1} {
		minimum := minimum
		t.Run("", func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(Config{Min: minimum}, nil)
			if err == nil {
				t.Fatalf("Resolve minimum %v succeeded, want error", minimum)
			}
		})
	}
}

func TestResolveUsesDefaultMinimumForZero(t *testing.T) {
	t.Parallel()

	if _, err := Resolve(Config{Min: 0}, nil); err != nil {
		t.Fatalf("Resolve default minimum: %v", err)
	}
}

func TestResolveRejectsNonFiniteOverrideMinimums(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Config{
		Overrides: []domain.Rule{{Pattern: "**/critical/**", Min: math.NaN()}},
	}, nil)
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
		argument := argument
		t.Run(argument, func(t *testing.T) {
			t.Parallel()

			_, err := Resolve(Config{TestArgs: []string{argument}}, nil)
			if err == nil {
				t.Fatalf("Resolve accepted reserved test argument %q, want error", argument)
			}
		})
	}
}
