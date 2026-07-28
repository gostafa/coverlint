package domain_test

import (
	"math"
	"testing"

	"github.com/gostafa/coverlint/internal/features/coverage/domain"
)

const (
	ordersFile    = "/repo/internal/orders/orders.go"
	ordersPackage = "github.com/acme/project/internal/orders"
	blockPosition = "10.1,12.2"
)

func TestNewPolicyRejectsNonFiniteMinimums(t *testing.T) {
	t.Parallel()

	for _, minimum := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -1, 100.1} {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewPolicy([]domain.Rule{{Pattern: "**", Min: minimum}}, nil)
			if err == nil {
				t.Fatalf("NewPolicy minimum %v succeeded, want error", minimum)
			}
		})
	}
}

func TestEvaluateDeduplicatesProfileBlocksByPosition(t *testing.T) {
	t.Parallel()

	policy := mustPolicy(t)
	report := policy.Evaluate([]domain.Package{{
		ImportPath: ordersPackage,
		Dir:        "/repo/internal/orders",
		Files:      []string{ordersFile},
		FirstFile:  ordersFile,
	}}, []domain.Block{
		{
			File:       ordersFile,
			Position:   blockPosition,
			Statements: 2,
			Covered:    false,
		},
		{
			File:       ordersFile,
			Position:   blockPosition,
			Statements: 2,
			Covered:    true,
		},
		{
			File:       ordersPackage + "/orders.go",
			Position:   blockPosition,
			Statements: 2,
			Covered:    false,
		},
	})

	if report.Failed != 0 || report.Checked != 1 {
		t.Fatalf("report = %#v, want one checked package with no failures", report)
	}

	result := report.Results[0]
	if result.Statements != 2 || result.Covered != 2 || result.Coverage != 100 {
		t.Fatalf("result = %#v, want duplicate block counted once as covered", result)
	}
}

func TestEvaluateDoesNotMapUnknownChildFileToAncestorPackage(t *testing.T) {
	t.Parallel()

	policy := mustPolicy(t)
	report := policy.Evaluate([]domain.Package{{
		ImportPath: ordersPackage,
		Dir:        "/repo/internal/orders",
		Files:      []string{ordersFile},
		FirstFile:  ordersFile,
	}}, []domain.Block{
		{
			File:       "/repo/internal/orders/subpkg/sub.go",
			Position:   "1.1,2.2",
			Statements: 1,
			Covered:    true,
		},
	})

	if report.Checked != 0 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("report = %#v, want unknown child file ignored and package skipped", report)
	}

	if !report.Results[0].Skipped {
		t.Fatalf("result = %#v, want skipped result", report.Results[0])
	}
}

func TestEvaluateSkipsPackageWithNoProfileBlocks(t *testing.T) {
	t.Parallel()

	policy := mustPolicy(t)
	report := policy.Evaluate([]domain.Package{{
		ImportPath: "github.com/acme/project/internal/contracts",
		Dir:        "/repo/internal/contracts",
		Files:      []string{"/repo/internal/contracts/contracts.go"},
		FirstFile:  "/repo/internal/contracts/contracts.go",
	}}, nil)

	if report.Checked != 0 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("report = %#v, want package skipped", report)
	}
}

func mustPolicy(t *testing.T) domain.Policy {
	t.Helper()

	policy, err := domain.NewPolicy([]domain.Rule{{Pattern: "**", Min: 80}}, nil)
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	return policy
}
