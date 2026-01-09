package domain

import (
	"testing"
	"time"
)

func TestTrendAnalysisService(t *testing.T) {
	t.Run("AnalyzeTrend returns stable for nil entries", func(t *testing.T) {
		service := NewTrendAnalysisService()

		result := service.AnalyzeTrend(nil, nil)

		if result.OverallTrend.Direction != TrendStable {
			t.Errorf("Expected stable trend, got %s", result.OverallTrend.Direction)
		}
	})

	t.Run("AnalyzeTrend detects improvement", func(t *testing.T) {
		service := NewTrendAnalysisService()

		previous := &HistoryEntry{
			Timestamp: time.Now().Add(-24 * time.Hour),
			Overall:   70.0,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 70.0},
			},
		}

		current := &HistoryEntry{
			Timestamp: time.Now(),
			Overall:   80.0,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 80.0},
			},
		}

		result := service.AnalyzeTrend(previous, current)

		if !result.IsImproving {
			t.Error("Expected IsImproving to be true")
		}
		if result.IsRegressing {
			t.Error("Expected IsRegressing to be false")
		}
		if result.OverallTrend.Direction != TrendUp {
			t.Errorf("Expected TrendUp, got %s", result.OverallTrend.Direction)
		}
		if result.OverallTrend.Delta != 10.0 {
			t.Errorf("Expected delta 10.0, got %v", result.OverallTrend.Delta)
		}
		if result.Current.Value() != 80.0 {
			t.Errorf("Expected current 80, got %v", result.Current.Value())
		}
		if result.Previous.Value() != 70.0 {
			t.Errorf("Expected previous 70, got %v", result.Previous.Value())
		}
	})

	t.Run("AnalyzeTrend detects regression", func(t *testing.T) {
		service := NewTrendAnalysisService()

		previous := &HistoryEntry{
			Timestamp: time.Now().Add(-24 * time.Hour),
			Overall:   85.0,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 85.0},
			},
		}

		current := &HistoryEntry{
			Timestamp: time.Now(),
			Overall:   75.0,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 75.0},
			},
		}

		result := service.AnalyzeTrend(previous, current)

		if result.IsImproving {
			t.Error("Expected IsImproving to be false")
		}
		if !result.IsRegressing {
			t.Error("Expected IsRegressing to be true")
		}
		if result.OverallTrend.Direction != TrendDown {
			t.Errorf("Expected TrendDown, got %s", result.OverallTrend.Direction)
		}
	})

	t.Run("AnalyzeTrend detects stable coverage", func(t *testing.T) {
		service := NewTrendAnalysisService()

		previous := &HistoryEntry{
			Timestamp: time.Now().Add(-24 * time.Hour),
			Overall:   80.0,
			Domains:   map[string]DomainEntry{},
		}

		current := &HistoryEntry{
			Timestamp: time.Now(),
			Overall:   80.3, // Within stability threshold
			Domains:   map[string]DomainEntry{},
		}

		result := service.AnalyzeTrend(previous, current)

		if result.OverallTrend.Direction != TrendStable {
			t.Errorf("Expected TrendStable, got %s", result.OverallTrend.Direction)
		}
	})

	t.Run("AnalyzeTrend calculates domain trends", func(t *testing.T) {
		service := NewTrendAnalysisService()

		previous := &HistoryEntry{
			Timestamp: time.Now().Add(-24 * time.Hour),
			Overall:   75.0,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 70.0},
				"api":  {Name: "api", Percent: 80.0},
			},
		}

		current := &HistoryEntry{
			Timestamp: time.Now(),
			Overall:   77.5,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 75.0}, // Improved
				"api":  {Name: "api", Percent: 80.0},  // Stable
			},
		}

		result := service.AnalyzeTrend(previous, current)

		if len(result.DomainTrends) != 2 {
			t.Fatalf("Expected 2 domain trends, got %d", len(result.DomainTrends))
		}

		coreTrend := result.DomainTrends["core"]
		if coreTrend.Trend.Direction != TrendUp {
			t.Errorf("Expected core TrendUp, got %s", coreTrend.Trend.Direction)
		}

		apiTrend := result.DomainTrends["api"]
		if apiTrend.Trend.Direction != TrendStable {
			t.Errorf("Expected api TrendStable, got %s", apiTrend.Trend.Direction)
		}
	})

	t.Run("AnalyzeTrend records events for significant changes", func(t *testing.T) {
		service := NewTrendAnalysisService()

		previous := &HistoryEntry{
			Timestamp: time.Now().Add(-24 * time.Hour),
			Overall:   70.0,
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 70.0},
			},
		}

		current := &HistoryEntry{
			Timestamp: time.Now(),
			Overall:   82.0, // >1% improvement
			Domains: map[string]DomainEntry{
				"core": {Name: "core", Percent: 82.0},
			},
		}

		service.AnalyzeTrend(previous, current)
		events := service.Events()

		hasImprovedEvent := false
		for _, e := range events {
			if e.EventType() == "CoverageImproved" {
				hasImprovedEvent = true
			}
		}

		if !hasImprovedEvent {
			t.Error("Expected CoverageImproved event for significant improvement")
		}
	})

	t.Run("ClearEvents removes all events", func(t *testing.T) {
		service := NewTrendAnalysisService()

		previous := &HistoryEntry{Overall: 70.0, Timestamp: time.Now().Add(-time.Hour)}
		current := &HistoryEntry{Overall: 82.0, Timestamp: time.Now()}

		service.AnalyzeTrend(previous, current)
		service.ClearEvents()

		if len(service.Events()) != 0 {
			t.Error("Expected no events after clear")
		}
	})
}

func TestHistoryAnalysis(t *testing.T) {
	t.Run("AnalyzeHistory returns stats for empty history", func(t *testing.T) {
		service := NewTrendAnalysisService()
		history := &History{Entries: []HistoryEntry{}}

		result := service.AnalyzeHistory(history, time.Now().Add(-7*24*time.Hour))

		if result.EntriesCount != 0 {
			t.Errorf("Expected 0 entries, got %d", result.EntriesCount)
		}
	})

	t.Run("AnalyzeHistory calculates correct statistics", func(t *testing.T) {
		service := NewTrendAnalysisService()

		now := time.Now()
		history := &History{
			Entries: []HistoryEntry{
				{Timestamp: now.Add(-6 * 24 * time.Hour), Overall: 70.0},
				{Timestamp: now.Add(-5 * 24 * time.Hour), Overall: 72.0}, // Up
				{Timestamp: now.Add(-4 * 24 * time.Hour), Overall: 71.5}, // Stable
				{Timestamp: now.Add(-3 * 24 * time.Hour), Overall: 75.0}, // Up
				{Timestamp: now.Add(-2 * 24 * time.Hour), Overall: 73.0}, // Down
				{Timestamp: now.Add(-1 * 24 * time.Hour), Overall: 80.0}, // Up
			},
		}

		result := service.AnalyzeHistory(history, now.Add(-7*24*time.Hour))

		if result.EntriesCount != 6 {
			t.Errorf("Expected 6 entries, got %d", result.EntriesCount)
		}
		if result.Highest.Value() != 80.0 {
			t.Errorf("Expected highest 80, got %v", result.Highest.Value())
		}
		if result.Lowest.Value() != 70.0 {
			t.Errorf("Expected lowest 70, got %v", result.Lowest.Value())
		}
		if result.UpDays != 3 {
			t.Errorf("Expected 3 up days, got %d", result.UpDays)
		}
		if result.DownDays != 1 {
			t.Errorf("Expected 1 down day, got %d", result.DownDays)
		}
		if result.StableDays != 1 {
			t.Errorf("Expected 1 stable day, got %d", result.StableDays)
		}
	})

	t.Run("Volatility returns correct value", func(t *testing.T) {
		result := HistoryAnalysisResult{
			EntriesCount: 5,
			UpDays:       2,
			DownDays:     1,
			StableDays:   1,
		}

		volatility := result.Volatility()
		expected := float64(3) / float64(4) // (2+1)/4 = 0.75
		if volatility != expected {
			t.Errorf("Expected volatility %v, got %v", expected, volatility)
		}
	})

	t.Run("ConsistencyScore returns correct value", func(t *testing.T) {
		result := HistoryAnalysisResult{
			EntriesCount: 5,
			Highest:      NewPercentage(85),
			Lowest:       NewPercentage(75),
		}

		score := result.ConsistencyScore()
		// Range is 10, so consistency = 100 - 10 = 90
		if score != 90.0 {
			t.Errorf("Expected consistency score 90, got %v", score)
		}
	})
}

func TestPredictNextCoverage(t *testing.T) {
	t.Run("PredictNextCoverage handles empty history", func(t *testing.T) {
		service := NewTrendAnalysisService()
		history := &History{Entries: []HistoryEntry{}}

		predicted, confidence := service.PredictNextCoverage(history, 5)

		if predicted.Value() != 0 {
			t.Errorf("Expected predicted 0, got %v", predicted.Value())
		}
		if confidence != 0 {
			t.Errorf("Expected confidence 0, got %v", confidence)
		}
	})

	t.Run("PredictNextCoverage handles single entry", func(t *testing.T) {
		service := NewTrendAnalysisService()
		history := &History{
			Entries: []HistoryEntry{
				{Overall: 80.0},
			},
		}

		predicted, confidence := service.PredictNextCoverage(history, 5)

		if predicted.Value() != 80.0 {
			t.Errorf("Expected predicted 80, got %v", predicted.Value())
		}
		if confidence != 0.5 {
			t.Errorf("Expected confidence 0.5, got %v", confidence)
		}
	})

	t.Run("PredictNextCoverage predicts upward trend", func(t *testing.T) {
		service := NewTrendAnalysisService()
		history := &History{
			Entries: []HistoryEntry{
				{Overall: 70.0},
				{Overall: 75.0},
				{Overall: 80.0},
				{Overall: 85.0},
			},
		}

		predicted, _ := service.PredictNextCoverage(history, 4)

		// With linear regression on 70, 75, 80, 85, next should be ~90
		if predicted.Value() < 85.0 {
			t.Errorf("Expected predicted > 85, got %v", predicted.Value())
		}
	})

	t.Run("PredictNextCoverage clamps to valid range", func(t *testing.T) {
		service := NewTrendAnalysisService()
		history := &History{
			Entries: []HistoryEntry{
				{Overall: 95.0},
				{Overall: 97.0},
				{Overall: 99.0},
			},
		}

		predicted, _ := service.PredictNextCoverage(history, 3)

		if predicted.Value() > 100 {
			t.Errorf("Expected predicted <= 100, got %v", predicted.Value())
		}
	})
}
