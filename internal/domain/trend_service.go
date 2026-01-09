package domain

import "time"

// TrendAnalysisService is a domain service that analyzes coverage trends
// over time and raises events when significant changes occur.
type TrendAnalysisService struct {
	events *EventCollector
}

// NewTrendAnalysisService creates a new TrendAnalysisService.
func NewTrendAnalysisService() *TrendAnalysisService {
	return &TrendAnalysisService{
		events: NewEventCollector(),
	}
}

// TrendAnalysisResult contains the results of trend analysis.
type TrendAnalysisResult struct {
	Current       Percentage
	Previous      Percentage
	OverallTrend  Trend
	DomainTrends  map[string]DomainTrendResult
	Period        time.Duration
	EntriesCount  int
	IsImproving   bool
	IsRegressing  bool
	StabilityDays int
}

// DomainTrendResult contains trend information for a single domain.
type DomainTrendResult struct {
	DomainName DomainName
	Current    Percentage
	Previous   Percentage
	Trend      Trend
}

// AnalyzeTrend analyzes the trend between a previous and current state.
func (s *TrendAnalysisService) AnalyzeTrend(previousEntry, currentEntry *HistoryEntry) TrendAnalysisResult {
	if previousEntry == nil || currentEntry == nil {
		return TrendAnalysisResult{
			OverallTrend: Trend{Direction: TrendStable, Delta: 0},
			DomainTrends: make(map[string]DomainTrendResult),
		}
	}

	current := NewPercentage(currentEntry.Overall)
	previous := NewPercentage(previousEntry.Overall)
	overallTrend := CalculateTrend(previous.Value(), current.Value())

	period := currentEntry.Timestamp.Sub(previousEntry.Timestamp)

	domainTrends := s.analyzeDomainTrends(previousEntry, currentEntry)

	result := TrendAnalysisResult{
		Current:      current,
		Previous:     previous,
		OverallTrend: overallTrend,
		DomainTrends: domainTrends,
		Period:       period,
		IsImproving:  overallTrend.Direction == TrendUp,
		IsRegressing: overallTrend.Direction == TrendDown,
	}

	// Record events for significant changes
	s.recordTrendEvents(result)

	return result
}

// analyzeDomainTrends analyzes trends for each domain.
func (s *TrendAnalysisService) analyzeDomainTrends(previous, current *HistoryEntry) map[string]DomainTrendResult {
	results := make(map[string]DomainTrendResult)

	for domainName, currentDomain := range current.Domains {
		currentPercent := NewPercentage(currentDomain.Percent)

		var previousPercent Percentage
		if prevDomain, ok := previous.Domains[domainName]; ok {
			previousPercent = NewPercentage(prevDomain.Percent)
		}

		trend := CalculateTrend(previousPercent.Value(), currentPercent.Value())

		domainNameObj, err := NewDomainName(domainName)
		if err != nil {
			continue
		}

		results[domainName] = DomainTrendResult{
			DomainName: domainNameObj,
			Current:    currentPercent,
			Previous:   previousPercent,
			Trend:      trend,
		}
	}

	return results
}

// recordTrendEvents records domain events for significant trend changes.
func (s *TrendAnalysisService) recordTrendEvents(result TrendAnalysisResult) {
	if result.IsImproving && result.OverallTrend.Delta > 1.0 {
		s.events.Record(NewCoverageImprovedEvent(
			"overall",
			result.Previous.Value(),
			result.Current.Value(),
		))
	}

	if result.IsRegressing && result.OverallTrend.Delta < -1.0 {
		s.events.Record(NewCoverageRegressedEvent(
			"overall",
			result.Previous.Value(),
			result.Current.Value(),
		))
	}

	for _, domainTrend := range result.DomainTrends {
		if domainTrend.Trend.Direction == TrendUp && domainTrend.Trend.Delta > 1.0 {
			s.events.Record(NewCoverageImprovedEvent(
				domainTrend.DomainName.String(),
				domainTrend.Previous.Value(),
				domainTrend.Current.Value(),
			))
		}
		if domainTrend.Trend.Direction == TrendDown && domainTrend.Trend.Delta < -1.0 {
			s.events.Record(NewCoverageRegressedEvent(
				domainTrend.DomainName.String(),
				domainTrend.Previous.Value(),
				domainTrend.Current.Value(),
			))
		}
	}
}

// AnalyzeHistory analyzes a sequence of history entries and returns trend statistics.
func (s *TrendAnalysisService) AnalyzeHistory(history *History, since time.Time) HistoryAnalysisResult {
	entries := history.EntriesAfter(since)
	if len(entries) == 0 {
		return HistoryAnalysisResult{}
	}

	var (
		highest     float64
		lowest      float64 = 100
		sum         float64
		upDays      int
		downDays    int
		stableDays  int
		prevPercent float64
	)

	for i, entry := range entries {
		if entry.Overall > highest {
			highest = entry.Overall
		}
		if entry.Overall < lowest {
			lowest = entry.Overall
		}
		sum += entry.Overall

		if i > 0 {
			delta := entry.Overall - prevPercent
			switch {
			case delta > 0.5:
				upDays++
			case delta < -0.5:
				downDays++
			default:
				stableDays++
			}
		}
		prevPercent = entry.Overall
	}

	avg := sum / float64(len(entries))

	return HistoryAnalysisResult{
		EntriesCount: len(entries),
		Highest:      NewPercentage(highest),
		Lowest:       NewPercentage(lowest),
		Average:      NewPercentage(avg),
		UpDays:       upDays,
		DownDays:     downDays,
		StableDays:   stableDays,
		Period:       time.Since(since),
	}
}

// HistoryAnalysisResult contains statistics about coverage history.
type HistoryAnalysisResult struct {
	EntriesCount int
	Highest      Percentage
	Lowest       Percentage
	Average      Percentage
	UpDays       int
	DownDays     int
	StableDays   int
	Period       time.Duration
}

// Volatility returns a measure of coverage volatility (0-1).
// Higher values indicate more volatile coverage.
func (r HistoryAnalysisResult) Volatility() float64 {
	if r.EntriesCount <= 1 {
		return 0
	}
	totalDays := r.UpDays + r.DownDays + r.StableDays
	if totalDays == 0 {
		return 0
	}
	return float64(r.UpDays+r.DownDays) / float64(totalDays)
}

// ConsistencyScore returns a score indicating how consistent coverage has been.
// Higher values indicate more consistent coverage (0-100).
func (r HistoryAnalysisResult) ConsistencyScore() float64 {
	if r.EntriesCount <= 1 {
		return 100
	}
	range_ := r.Highest.Value() - r.Lowest.Value()
	// Normalize: range of 0 = 100% consistent, range of 100 = 0% consistent
	return Round1(100 - range_)
}

// Events returns all domain events that were recorded during analysis.
func (s *TrendAnalysisService) Events() []DomainEvent {
	return s.events.Events()
}

// ClearEvents clears all recorded events.
func (s *TrendAnalysisService) ClearEvents() {
	s.events.Clear()
}

// PredictNextCoverage provides a simple linear prediction based on recent trend.
// Returns the predicted coverage and confidence level.
func (s *TrendAnalysisService) PredictNextCoverage(history *History, lookbackEntries int) (predicted Percentage, confidence float64) {
	if len(history.Entries) < 2 {
		if len(history.Entries) == 1 {
			return NewPercentage(history.Entries[0].Overall), 0.5
		}
		return NewPercentage(0), 0
	}

	start := len(history.Entries) - lookbackEntries
	if start < 0 {
		start = 0
	}
	entries := history.Entries[start:]

	if len(entries) < 2 {
		return NewPercentage(entries[len(entries)-1].Overall), 0.5
	}

	// Simple linear regression
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(entries))

	for i, entry := range entries {
		x := float64(i)
		y := entry.Overall
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Predict next value (x = n)
	nextX := n
	predictedValue := slope*nextX + intercept

	// Clamp to valid range
	if predictedValue < 0 {
		predictedValue = 0
	}
	if predictedValue > 100 {
		predictedValue = 100
	}

	// Confidence based on variance
	var variance float64
	for i, entry := range entries {
		expected := slope*float64(i) + intercept
		diff := entry.Overall - expected
		variance += diff * diff
	}
	variance /= n

	// Convert variance to confidence (lower variance = higher confidence)
	// Using exponential decay: confidence = e^(-variance/100)
	confidence = 1.0 / (1.0 + variance/100)

	return NewPercentage(predictedValue), Round1(confidence * 100)
}
