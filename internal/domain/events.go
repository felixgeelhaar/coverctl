package domain

import "time"

// DomainEvent represents a significant occurrence in the domain.
type DomainEvent interface {
	// OccurredAt returns when the event occurred.
	OccurredAt() time.Time
	// EventType returns the type of event.
	EventType() string
}

// BaseEvent provides common event functionality.
type BaseEvent struct {
	occurredAt time.Time
}

// OccurredAt returns when the event occurred.
func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// NewBaseEvent creates a new base event with current timestamp.
func NewBaseEvent() BaseEvent {
	return BaseEvent{occurredAt: time.Now()}
}

// CoverageEvaluatedEvent is raised when coverage is evaluated against a policy.
type CoverageEvaluatedEvent struct {
	BaseEvent
	PolicyName     string
	OverallPercent float64
	Passed         bool
	DomainCount    int
	FailedCount    int
}

// EventType returns the event type identifier.
func (e CoverageEvaluatedEvent) EventType() string {
	return "CoverageEvaluated"
}

// NewCoverageEvaluatedEvent creates a new CoverageEvaluatedEvent.
func NewCoverageEvaluatedEvent(policyName string, overallPercent float64, passed bool, domainCount, failedCount int) CoverageEvaluatedEvent {
	return CoverageEvaluatedEvent{
		BaseEvent:      NewBaseEvent(),
		PolicyName:     policyName,
		OverallPercent: overallPercent,
		Passed:         passed,
		DomainCount:    domainCount,
		FailedCount:    failedCount,
	}
}

// ThresholdViolatedEvent is raised when coverage falls below a threshold.
type ThresholdViolatedEvent struct {
	BaseEvent
	DomainName string
	Actual     float64
	Required   float64
	Shortfall  float64
}

// EventType returns the event type identifier.
func (e ThresholdViolatedEvent) EventType() string {
	return "ThresholdViolated"
}

// NewThresholdViolatedEvent creates a new ThresholdViolatedEvent.
func NewThresholdViolatedEvent(domainName string, actual, required float64) ThresholdViolatedEvent {
	return ThresholdViolatedEvent{
		BaseEvent:  NewBaseEvent(),
		DomainName: domainName,
		Actual:     actual,
		Required:   required,
		Shortfall:  Round1(required - actual),
	}
}

// CoverageImprovedEvent is raised when coverage improves.
type CoverageImprovedEvent struct {
	BaseEvent
	DomainName string
	Previous   float64
	Current    float64
	Delta      float64
}

// EventType returns the event type identifier.
func (e CoverageImprovedEvent) EventType() string {
	return "CoverageImproved"
}

// NewCoverageImprovedEvent creates a new CoverageImprovedEvent.
func NewCoverageImprovedEvent(domainName string, previous, current float64) CoverageImprovedEvent {
	return CoverageImprovedEvent{
		BaseEvent:  NewBaseEvent(),
		DomainName: domainName,
		Previous:   previous,
		Current:    current,
		Delta:      Round1(current - previous),
	}
}

// CoverageRegressedEvent is raised when coverage decreases.
type CoverageRegressedEvent struct {
	BaseEvent
	DomainName string
	Previous   float64
	Current    float64
	Delta      float64
}

// EventType returns the event type identifier.
func (e CoverageRegressedEvent) EventType() string {
	return "CoverageRegressed"
}

// NewCoverageRegressedEvent creates a new CoverageRegressedEvent.
func NewCoverageRegressedEvent(domainName string, previous, current float64) CoverageRegressedEvent {
	return CoverageRegressedEvent{
		BaseEvent:  NewBaseEvent(),
		DomainName: domainName,
		Previous:   previous,
		Current:    current,
		Delta:      Round1(previous - current),
	}
}

// EventPublisher publishes domain events.
type EventPublisher interface {
	Publish(event DomainEvent) error
	PublishAll(events []DomainEvent) error
}

// EventCollector collects domain events for later publishing.
type EventCollector struct {
	events []DomainEvent
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]DomainEvent, 0),
	}
}

// Record adds an event to the collector.
func (c *EventCollector) Record(event DomainEvent) {
	c.events = append(c.events, event)
}

// Events returns all collected events.
func (c *EventCollector) Events() []DomainEvent {
	return c.events
}

// Clear removes all collected events.
func (c *EventCollector) Clear() {
	c.events = make([]DomainEvent, 0)
}

// HasEvents returns true if there are any collected events.
func (c *EventCollector) HasEvents() bool {
	return len(c.events) > 0
}
