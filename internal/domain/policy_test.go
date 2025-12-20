package domain

import "testing"

func TestEvaluatePolicy(t *testing.T) {
	min := 85.0
	policy := Policy{
		DefaultMin: 80,
		Domains: []Domain{
			{Name: "core", Min: &min},
			{Name: "api"},
		},
	}
	coverage := map[string]CoverageStat{
		"core": {Covered: 16, Total: 20},
		"api":  {Covered: 8, Total: 10},
	}

	result := Evaluate(policy, coverage)
	if result.Passed {
		t.Fatalf("expected policy to fail")
	}
	if got := result.Domains[0].Status; got != StatusFail {
		t.Fatalf("expected core to fail, got %s", got)
	}
	if got := result.Domains[1].Status; got != StatusPass {
		t.Fatalf("expected api to pass, got %s", got)
	}
}

func TestCoveragePercent(t *testing.T) {
	stat := CoverageStat{Covered: 1, Total: 3}
	if got := stat.Percent(); got < 33.3 || got > 33.4 {
		t.Fatalf("expected ~33.3, got %f", got)
	}
	zero := CoverageStat{}
	if got := zero.Percent(); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}
