package domain

import (
	"testing"
)

func TestPolicyAggregate(t *testing.T) {
	t.Run("NewPolicyAggregate creates aggregate from policy", func(t *testing.T) {
		min80 := 80.0
		min90 := 90.0
		policy := Policy{
			DefaultMin: 70,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
				{Name: "api", Match: []string{"internal/api/**"}, Min: &min90},
				{Name: "utils", Match: []string{"internal/utils/**"}}, // Uses default
			},
		}

		agg, err := NewPolicyAggregate(policy)
		if err != nil {
			t.Fatalf("NewPolicyAggregate failed: %v", err)
		}

		if agg.DefaultMin().Value() != 70 {
			t.Errorf("Expected default min 70, got %v", agg.DefaultMin().Value())
		}

		specs := agg.DomainSpecs()
		if len(specs) != 3 {
			t.Fatalf("Expected 3 domain specs, got %d", len(specs))
		}

		if specs[0].Name.String() != "core" {
			t.Errorf("Expected first domain 'core', got '%s'", specs[0].Name.String())
		}
		if specs[0].MinValue.Value() != 80 {
			t.Errorf("Expected core min 80, got %v", specs[0].MinValue.Value())
		}

		if specs[2].MinValue.Value() != 70 {
			t.Errorf("Expected utils to use default min 70, got %v", specs[2].MinValue.Value())
		}
	})

	t.Run("NewPolicyAggregate rejects invalid threshold", func(t *testing.T) {
		policy := Policy{
			DefaultMin: 150, // Invalid
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}},
			},
		}

		_, err := NewPolicyAggregate(policy)
		if err == nil {
			t.Error("Expected error for invalid threshold")
		}
	})

	t.Run("NewPolicyAggregate rejects empty domain name", func(t *testing.T) {
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "", Match: []string{"internal/core/**"}},
			},
		}

		_, err := NewPolicyAggregate(policy)
		if err == nil {
			t.Error("Expected error for empty domain name")
		}
	})

	t.Run("Evaluate returns correct results for passing domains", func(t *testing.T) {
		min80 := 80.0
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
				{Name: "api", Match: []string{"internal/api/**"}},
			},
		}

		agg, _ := NewPolicyAggregate(policy)

		coverage := map[string]CoverageStat{
			"core": {Covered: 85, Total: 100},
			"api":  {Covered: 90, Total: 100},
		}

		result := agg.Evaluate(coverage)

		if !result.Passed {
			t.Error("Expected all domains to pass")
		}
		if len(result.DomainResults) != 2 {
			t.Fatalf("Expected 2 domain results, got %d", len(result.DomainResults))
		}

		coreResult := result.DomainResults[0]
		if coreResult.Status != StatusPass {
			t.Errorf("Expected core to pass, got status %s", coreResult.Status)
		}
		if coreResult.Percent.Value() != 85.0 {
			t.Errorf("Expected core percent 85, got %v", coreResult.Percent.Value())
		}
	})

	t.Run("Evaluate returns correct results for failing domains", func(t *testing.T) {
		min80 := 80.0
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
				{Name: "api", Match: []string{"internal/api/**"}},
			},
		}

		agg, _ := NewPolicyAggregate(policy)

		coverage := map[string]CoverageStat{
			"core": {Covered: 70, Total: 100}, // Below threshold
			"api":  {Covered: 90, Total: 100},
		}

		result := agg.Evaluate(coverage)

		if result.Passed {
			t.Error("Expected evaluation to fail")
		}

		coreResult := result.DomainResults[0]
		if coreResult.Status != StatusFail {
			t.Errorf("Expected core to fail, got status %s", coreResult.Status)
		}
		if coreResult.Shortfall != 10.0 {
			t.Errorf("Expected shortfall 10, got %v", coreResult.Shortfall)
		}
	})

	t.Run("Evaluate handles warn threshold", func(t *testing.T) {
		min70 := 70.0
		warn80 := 80.0
		policy := Policy{
			DefaultMin: 70,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min70, Warn: &warn80},
			},
		}

		agg, _ := NewPolicyAggregate(policy)

		coverage := map[string]CoverageStat{
			"core": {Covered: 75, Total: 100}, // Above min, below warn
		}

		result := agg.Evaluate(coverage)

		if !result.Passed {
			t.Error("Expected evaluation to pass (above min)")
		}

		coreResult := result.DomainResults[0]
		if coreResult.Status != StatusWarn {
			t.Errorf("Expected core to warn, got status %s", coreResult.Status)
		}
	})

	t.Run("Evaluate records threshold violated events", func(t *testing.T) {
		min80 := 80.0
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
			},
		}

		agg, _ := NewPolicyAggregate(policy)

		coverage := map[string]CoverageStat{
			"core": {Covered: 70, Total: 100},
		}

		agg.Evaluate(coverage)
		events := agg.Events()

		hasViolation := false
		for _, e := range events {
			if e.EventType() == "ThresholdViolated" {
				hasViolation = true
				violated := e.(ThresholdViolatedEvent)
				if violated.DomainName != "core" {
					t.Errorf("Expected domain 'core', got '%s'", violated.DomainName)
				}
			}
		}

		if !hasViolation {
			t.Error("Expected ThresholdViolated event")
		}
	})

	t.Run("Evaluate records coverage evaluated event", func(t *testing.T) {
		min80 := 80.0
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
			},
		}

		agg, _ := NewPolicyAggregate(policy)

		coverage := map[string]CoverageStat{
			"core": {Covered: 85, Total: 100},
		}

		agg.Evaluate(coverage)
		events := agg.Events()

		hasEvaluated := false
		for _, e := range events {
			if e.EventType() == "CoverageEvaluated" {
				hasEvaluated = true
			}
		}

		if !hasEvaluated {
			t.Error("Expected CoverageEvaluated event")
		}
	})

	t.Run("ClearEvents removes all events", func(t *testing.T) {
		min80 := 80.0
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
			},
		}

		agg, _ := NewPolicyAggregate(policy)

		coverage := map[string]CoverageStat{
			"core": {Covered: 70, Total: 100},
		}

		agg.Evaluate(coverage)
		agg.ClearEvents()

		if len(agg.Events()) != 0 {
			t.Error("Expected no events after clear")
		}
	})
}

func TestEvaluationResult(t *testing.T) {
	t.Run("OverallPercent calculates correctly", func(t *testing.T) {
		result := EvaluationResult{
			DomainResults: []DomainEvaluationResult{
				{Stat: CoverageStat{Covered: 80, Total: 100}},
				{Stat: CoverageStat{Covered: 60, Total: 100}},
			},
		}

		overall := result.OverallPercent()
		if overall.Value() != 70.0 {
			t.Errorf("Expected 70%%, got %v", overall.Value())
		}
	})

	t.Run("FailingCount returns correct count", func(t *testing.T) {
		result := EvaluationResult{
			DomainResults: []DomainEvaluationResult{
				{Status: StatusPass},
				{Status: StatusFail},
				{Status: StatusFail},
				{Status: StatusWarn},
			},
		}

		if result.FailingCount() != 2 {
			t.Errorf("Expected 2 failing, got %d", result.FailingCount())
		}
	})

	t.Run("PassingCount returns correct count", func(t *testing.T) {
		result := EvaluationResult{
			DomainResults: []DomainEvaluationResult{
				{Status: StatusPass},
				{Status: StatusPass},
				{Status: StatusFail},
			},
		}

		if result.PassingCount() != 2 {
			t.Errorf("Expected 2 passing, got %d", result.PassingCount())
		}
	})

	t.Run("ToResult converts to legacy Result", func(t *testing.T) {
		result := EvaluationResult{
			DomainResults: []DomainEvaluationResult{
				{
					Name:     MustDomainName("core"),
					Stat:     CoverageStat{Covered: 80, Total: 100},
					Percent:  NewPercentage(80),
					Required: MustThreshold(80),
					Status:   StatusPass,
				},
			},
			Passed: true,
		}

		legacy := result.ToResult()

		if !legacy.Passed {
			t.Error("Expected Passed to be true")
		}
		if len(legacy.Domains) != 1 {
			t.Fatalf("Expected 1 domain, got %d", len(legacy.Domains))
		}
		if legacy.Domains[0].Domain != "core" {
			t.Errorf("Expected domain 'core', got '%s'", legacy.Domains[0].Domain)
		}
		if legacy.Domains[0].Percent != 80 {
			t.Errorf("Expected percent 80, got %v", legacy.Domains[0].Percent)
		}
	})
}

func TestEvaluateWithAggregate(t *testing.T) {
	t.Run("EvaluateWithAggregate returns result and events", func(t *testing.T) {
		min80 := 80.0
		policy := Policy{
			DefaultMin: 80,
			Domains: []Domain{
				{Name: "core", Match: []string{"internal/core/**"}, Min: &min80},
			},
		}

		coverage := map[string]CoverageStat{
			"core": {Covered: 85, Total: 100},
		}

		result, events, err := EvaluateWithAggregate(policy, coverage)
		if err != nil {
			t.Fatalf("EvaluateWithAggregate failed: %v", err)
		}

		if !result.Passed {
			t.Error("Expected result to pass")
		}

		if len(events) == 0 {
			t.Error("Expected at least one event")
		}
	})
}
