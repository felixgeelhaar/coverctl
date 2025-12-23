package domain

import (
	"testing"
	"time"
)

func TestHistoryLatestEntry(t *testing.T) {
	t.Run("empty history returns nil", func(t *testing.T) {
		h := History{}
		if h.LatestEntry() != nil {
			t.Fatal("expected nil for empty history")
		}
	})

	t.Run("returns last entry", func(t *testing.T) {
		h := History{
			Entries: []HistoryEntry{
				{Timestamp: time.Now().Add(-2 * time.Hour), Overall: 70.0},
				{Timestamp: time.Now().Add(-1 * time.Hour), Overall: 75.0},
				{Timestamp: time.Now(), Overall: 80.0},
			},
		}
		latest := h.LatestEntry()
		if latest == nil {
			t.Fatal("expected non-nil entry")
		}
		if latest.Overall != 80.0 {
			t.Fatalf("expected 80.0, got %f", latest.Overall)
		}
	})
}

func TestHistoryEntriesAfter(t *testing.T) {
	now := time.Now()
	h := History{
		Entries: []HistoryEntry{
			{Timestamp: now.Add(-3 * time.Hour), Overall: 70.0},
			{Timestamp: now.Add(-2 * time.Hour), Overall: 72.0},
			{Timestamp: now.Add(-1 * time.Hour), Overall: 75.0},
			{Timestamp: now, Overall: 80.0},
		},
	}

	t.Run("returns entries after cutoff", func(t *testing.T) {
		entries := h.EntriesAfter(now.Add(-90 * time.Minute))
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("returns empty for future cutoff", func(t *testing.T) {
		entries := h.EntriesAfter(now.Add(time.Hour))
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
	})
}

func TestCalculateTrend(t *testing.T) {
	tests := []struct {
		name      string
		previous  float64
		current   float64
		wantDir   TrendDirection
		wantDelta float64
	}{
		{"significant increase", 70.0, 75.0, TrendUp, 5.0},
		{"significant decrease", 80.0, 75.0, TrendDown, -5.0},
		{"stable - no change", 75.0, 75.0, TrendStable, 0.0},
		{"stable - small increase", 75.0, 75.4, TrendStable, 0.4},
		{"stable - small decrease", 75.0, 74.6, TrendStable, -0.4},
		{"just above threshold", 70.0, 70.6, TrendUp, 0.6},
		{"just below threshold", 70.0, 69.4, TrendDown, -0.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := CalculateTrend(tt.previous, tt.current)
			if trend.Direction != tt.wantDir {
				t.Errorf("direction = %v, want %v", trend.Direction, tt.wantDir)
			}
			// Use approximate comparison for floating-point values
			if diff := trend.Delta - tt.wantDelta; diff > 0.001 || diff < -0.001 {
				t.Errorf("delta = %v, want %v", trend.Delta, tt.wantDelta)
			}
		})
	}
}
