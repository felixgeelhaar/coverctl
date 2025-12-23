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

func TestEvaluateWarnThreshold(t *testing.T) {
	min := 80.0
	warn := 90.0
	policy := Policy{
		DefaultMin: 75,
		Domains: []Domain{
			{Name: "core", Min: &min, Warn: &warn},
		},
	}
	// Coverage is above min but below warn
	coverage := map[string]CoverageStat{
		"core": {Covered: 85, Total: 100},
	}

	result := Evaluate(policy, coverage)
	if !result.Passed {
		t.Fatal("expected policy to pass (above min)")
	}
	if result.Domains[0].Status != StatusWarn {
		t.Fatalf("expected warn status, got %s", result.Domains[0].Status)
	}
}

func TestEvaluateWarnThresholdPassAboveWarn(t *testing.T) {
	min := 80.0
	warn := 85.0
	policy := Policy{
		DefaultMin: 75,
		Domains: []Domain{
			{Name: "core", Min: &min, Warn: &warn},
		},
	}
	// Coverage is above both min and warn
	coverage := map[string]CoverageStat{
		"core": {Covered: 90, Total: 100},
	}

	result := Evaluate(policy, coverage)
	if !result.Passed {
		t.Fatal("expected policy to pass")
	}
	if result.Domains[0].Status != StatusPass {
		t.Fatalf("expected pass status, got %s", result.Domains[0].Status)
	}
}

func TestEvaluateWarnThresholdFailBelowMin(t *testing.T) {
	min := 80.0
	warn := 90.0
	policy := Policy{
		DefaultMin: 75,
		Domains: []Domain{
			{Name: "core", Min: &min, Warn: &warn},
		},
	}
	// Coverage is below min
	coverage := map[string]CoverageStat{
		"core": {Covered: 70, Total: 100},
	}

	result := Evaluate(policy, coverage)
	if result.Passed {
		t.Fatal("expected policy to fail (below min)")
	}
	if result.Domains[0].Status != StatusFail {
		t.Fatalf("expected fail status, got %s", result.Domains[0].Status)
	}
}

func TestEvaluateNoWarnThreshold(t *testing.T) {
	min := 80.0
	policy := Policy{
		DefaultMin: 75,
		Domains: []Domain{
			{Name: "core", Min: &min}, // No warn set
		},
	}
	// Coverage is above min - should just pass without warn
	coverage := map[string]CoverageStat{
		"core": {Covered: 85, Total: 100},
	}

	result := Evaluate(policy, coverage)
	if !result.Passed {
		t.Fatal("expected policy to pass")
	}
	if result.Domains[0].Status != StatusPass {
		t.Fatalf("expected pass status, got %s", result.Domains[0].Status)
	}
}
