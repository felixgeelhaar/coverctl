package domain

import "time"

// HistoryEntry represents a single coverage measurement over time.
type HistoryEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Commit    string                 `json:"commit,omitempty"`
	Branch    string                 `json:"branch,omitempty"`
	Overall   float64                `json:"overall"`
	Domains   map[string]DomainEntry `json:"domains"`
}

// DomainEntry represents coverage for a single domain at a point in time.
type DomainEntry struct {
	Name    string  `json:"name"`
	Percent float64 `json:"percent"`
	Min     float64 `json:"min"`
	Status  Status  `json:"status"`
}

// Trend represents the direction and magnitude of coverage change.
type Trend struct {
	Direction TrendDirection `json:"direction"`
	Delta     float64        `json:"delta"`
	Period    string         `json:"period"`
}

// TrendDirection indicates whether coverage is improving, declining, or stable.
type TrendDirection string

const (
	TrendUp     TrendDirection = "up"
	TrendDown   TrendDirection = "down"
	TrendStable TrendDirection = "stable"
)

// History contains all historical coverage entries.
type History struct {
	Entries []HistoryEntry `json:"entries"`
}

// LatestEntry returns the most recent history entry, or nil if empty.
func (h *History) LatestEntry() *HistoryEntry {
	if len(h.Entries) == 0 {
		return nil
	}
	latestIndex := 0
	latestTime := h.Entries[0].Timestamp
	for i := 1; i < len(h.Entries); i++ {
		if h.Entries[i].Timestamp.After(latestTime) {
			latestIndex = i
			latestTime = h.Entries[i].Timestamp
		}
	}
	return &h.Entries[latestIndex]
}

// EntriesAfter returns all entries after the given time.
func (h *History) EntriesAfter(t time.Time) []HistoryEntry {
	var result []HistoryEntry
	for _, e := range h.Entries {
		if e.Timestamp.After(t) {
			result = append(result, e)
		}
	}
	return result
}

// CalculateTrend computes the trend between two coverage values.
func CalculateTrend(previous, current float64) Trend {
	delta := current - previous
	var direction TrendDirection

	switch {
	case delta > 0.5:
		direction = TrendUp
	case delta < -0.5:
		direction = TrendDown
	default:
		direction = TrendStable
	}

	return Trend{
		Direction: direction,
		Delta:     delta,
	}
}
