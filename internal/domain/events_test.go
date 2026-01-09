package domain

import (
	"testing"
	"time"
)

func TestBaseEvent(t *testing.T) {
	t.Run("NewBaseEvent sets timestamp", func(t *testing.T) {
		before := time.Now()
		event := NewBaseEvent()
		after := time.Now()

		if event.OccurredAt().Before(before) || event.OccurredAt().After(after) {
			t.Error("OccurredAt should be between before and after")
		}
	})
}

func TestCoverageEvaluatedEvent(t *testing.T) {
	t.Run("NewCoverageEvaluatedEvent creates event with correct values", func(t *testing.T) {
		event := NewCoverageEvaluatedEvent("default", 85.5, true, 3, 0)

		if event.EventType() != "CoverageEvaluated" {
			t.Errorf("Expected EventType 'CoverageEvaluated', got '%s'", event.EventType())
		}
		if event.PolicyName != "default" {
			t.Errorf("Expected PolicyName 'default', got '%s'", event.PolicyName)
		}
		if event.OverallPercent != 85.5 {
			t.Errorf("Expected OverallPercent 85.5, got %v", event.OverallPercent)
		}
		if !event.Passed {
			t.Error("Expected Passed to be true")
		}
		if event.DomainCount != 3 {
			t.Errorf("Expected DomainCount 3, got %d", event.DomainCount)
		}
		if event.FailedCount != 0 {
			t.Errorf("Expected FailedCount 0, got %d", event.FailedCount)
		}
	})
}

func TestThresholdViolatedEvent(t *testing.T) {
	t.Run("NewThresholdViolatedEvent creates event with correct values", func(t *testing.T) {
		event := NewThresholdViolatedEvent("core", 75.0, 80.0)

		if event.EventType() != "ThresholdViolated" {
			t.Errorf("Expected EventType 'ThresholdViolated', got '%s'", event.EventType())
		}
		if event.DomainName != "core" {
			t.Errorf("Expected DomainName 'core', got '%s'", event.DomainName)
		}
		if event.Actual != 75.0 {
			t.Errorf("Expected Actual 75.0, got %v", event.Actual)
		}
		if event.Required != 80.0 {
			t.Errorf("Expected Required 80.0, got %v", event.Required)
		}
		if event.Shortfall != 5.0 {
			t.Errorf("Expected Shortfall 5.0, got %v", event.Shortfall)
		}
	})
}

func TestCoverageImprovedEvent(t *testing.T) {
	t.Run("NewCoverageImprovedEvent creates event with correct values", func(t *testing.T) {
		event := NewCoverageImprovedEvent("core", 80.0, 85.0)

		if event.EventType() != "CoverageImproved" {
			t.Errorf("Expected EventType 'CoverageImproved', got '%s'", event.EventType())
		}
		if event.DomainName != "core" {
			t.Errorf("Expected DomainName 'core', got '%s'", event.DomainName)
		}
		if event.Previous != 80.0 {
			t.Errorf("Expected Previous 80.0, got %v", event.Previous)
		}
		if event.Current != 85.0 {
			t.Errorf("Expected Current 85.0, got %v", event.Current)
		}
		if event.Delta != 5.0 {
			t.Errorf("Expected Delta 5.0, got %v", event.Delta)
		}
	})
}

func TestCoverageRegressedEvent(t *testing.T) {
	t.Run("NewCoverageRegressedEvent creates event with correct values", func(t *testing.T) {
		event := NewCoverageRegressedEvent("core", 85.0, 80.0)

		if event.EventType() != "CoverageRegressed" {
			t.Errorf("Expected EventType 'CoverageRegressed', got '%s'", event.EventType())
		}
		if event.DomainName != "core" {
			t.Errorf("Expected DomainName 'core', got '%s'", event.DomainName)
		}
		if event.Previous != 85.0 {
			t.Errorf("Expected Previous 85.0, got %v", event.Previous)
		}
		if event.Current != 80.0 {
			t.Errorf("Expected Current 80.0, got %v", event.Current)
		}
		if event.Delta != 5.0 {
			t.Errorf("Expected Delta 5.0, got %v", event.Delta)
		}
	})
}

func TestEventCollector(t *testing.T) {
	t.Run("NewEventCollector creates empty collector", func(t *testing.T) {
		collector := NewEventCollector()

		if collector.HasEvents() {
			t.Error("New collector should have no events")
		}
		if len(collector.Events()) != 0 {
			t.Error("New collector should have empty events slice")
		}
	})

	t.Run("Record adds events", func(t *testing.T) {
		collector := NewEventCollector()

		event1 := NewCoverageEvaluatedEvent("policy1", 80.0, true, 1, 0)
		event2 := NewThresholdViolatedEvent("core", 75.0, 80.0)

		collector.Record(event1)
		collector.Record(event2)

		if !collector.HasEvents() {
			t.Error("Collector should have events after recording")
		}
		if len(collector.Events()) != 2 {
			t.Errorf("Expected 2 events, got %d", len(collector.Events()))
		}
	})

	t.Run("Clear removes all events", func(t *testing.T) {
		collector := NewEventCollector()

		collector.Record(NewCoverageEvaluatedEvent("policy1", 80.0, true, 1, 0))
		collector.Record(NewThresholdViolatedEvent("core", 75.0, 80.0))

		collector.Clear()

		if collector.HasEvents() {
			t.Error("Collector should have no events after clear")
		}
		if len(collector.Events()) != 0 {
			t.Error("Events slice should be empty after clear")
		}
	})

	t.Run("Events returns correct types", func(t *testing.T) {
		collector := NewEventCollector()

		collector.Record(NewCoverageEvaluatedEvent("policy1", 80.0, true, 1, 0))
		collector.Record(NewThresholdViolatedEvent("core", 75.0, 80.0))
		collector.Record(NewCoverageImprovedEvent("api", 70.0, 80.0))
		collector.Record(NewCoverageRegressedEvent("db", 90.0, 85.0))

		events := collector.Events()
		if len(events) != 4 {
			t.Fatalf("Expected 4 events, got %d", len(events))
		}

		if events[0].EventType() != "CoverageEvaluated" {
			t.Errorf("Expected first event to be CoverageEvaluated, got %s", events[0].EventType())
		}
		if events[1].EventType() != "ThresholdViolated" {
			t.Errorf("Expected second event to be ThresholdViolated, got %s", events[1].EventType())
		}
		if events[2].EventType() != "CoverageImproved" {
			t.Errorf("Expected third event to be CoverageImproved, got %s", events[2].EventType())
		}
		if events[3].EventType() != "CoverageRegressed" {
			t.Errorf("Expected fourth event to be CoverageRegressed, got %s", events[3].EventType())
		}
	})
}
