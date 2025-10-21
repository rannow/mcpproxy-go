package events

import (
	"sync"
	"time"

	"mcpproxy-go/internal/upstream/types"
)

// EventType defines the different types of events
type EventType int

const (
	// EventStateChange is fired when a server's connection state changes
	EventStateChange EventType = iota
	// EventConfigChange is fired when a server's configuration changes
	EventConfigChange
	// EventToolsDiscovered is fired when tools are discovered for a server
	EventToolsDiscovered
)

// String returns the string representation of the event type
func (t EventType) String() string {
	switch t {
	case EventStateChange:
		return "StateChange"
	case EventConfigChange:
		return "ConfigChange"
	case EventToolsDiscovered:
		return "ToolsDiscovered"
	default:
		return "Unknown"
	}
}

// Event is the base structure for all events
type Event struct {
	Type       EventType   `json:"type"`
	ServerName string      `json:"server_name"`
	Data       interface{} `json:"data"`
	Timestamp  time.Time   `json:"timestamp"`
}

// StateChangeData contains data for state change events
type StateChangeData struct {
	OldState types.ConnectionState `json:"old_state"`
	NewState types.ConnectionState `json:"new_state"`
	Info     *types.ConnectionInfo `json:"info"`
}

// ConfigChangeData contains data for configuration change events
type ConfigChangeData struct {
	Action string `json:"action"` // "enabled", "disabled", "quarantined", "unquarantined"
}

// ToolsDiscoveredData contains data for tools discovered events
type ToolsDiscoveredData struct {
	Count int `json:"count"`
}

// EventHandler is the callback function for event handling
type EventHandler func(event Event)

// EventBus manages event subscriptions and broadcasting
type EventBus struct {
	mu         sync.RWMutex
	handlers   map[EventType][]EventHandler
	debugMode  bool
	metrics    *EventMetrics
}

// EventMetrics tracks event bus statistics
type EventMetrics struct {
	mu              sync.RWMutex
	eventsPublished map[EventType]int64
	eventsHandled   map[EventType]int64
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
		metrics: &EventMetrics{
			eventsPublished: make(map[EventType]int64),
			eventsHandled:   make(map[EventType]int64),
		},
	}
}

// Subscribe registers a handler for an event type
// Handlers are called asynchronously in separate goroutines
func (bus *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// Publish sends an event to all registered handlers
// Broadcasting is done asynchronously to avoid blocking the caller
func (bus *EventBus) Publish(event Event) {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	bus.mu.RLock()
	handlers := bus.handlers[event.Type]
	bus.mu.RUnlock()

	// Update metrics
	bus.metrics.mu.Lock()
	bus.metrics.eventsPublished[event.Type]++
	bus.metrics.mu.Unlock()

	// Async broadcasting (non-blocking)
	for _, handler := range handlers {
		go func(h EventHandler, e Event) {
			h(e)
			// Update handled count
			bus.metrics.mu.Lock()
			bus.metrics.eventsHandled[e.Type]++
			bus.metrics.mu.Unlock()
		}(handler, event)
	}
}

// SetDebugMode enables or disables debug logging
func (bus *EventBus) SetDebugMode(enabled bool) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.debugMode = enabled
}

// GetMetrics returns a snapshot of current event metrics
func (bus *EventBus) GetMetrics() map[string]interface{} {
	bus.metrics.mu.RLock()
	defer bus.metrics.mu.RUnlock()

	metrics := make(map[string]interface{})
	published := make(map[string]int64)
	handled := make(map[string]int64)

	for eventType, count := range bus.metrics.eventsPublished {
		published[eventType.String()] = count
	}
	for eventType, count := range bus.metrics.eventsHandled {
		handled[eventType.String()] = count
	}

	metrics["published"] = published
	metrics["handled"] = handled

	return metrics
}

// Reset clears all handlers and metrics (useful for testing)
func (bus *EventBus) Reset() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.handlers = make(map[EventType][]EventHandler)
	bus.metrics = &EventMetrics{
		eventsPublished: make(map[EventType]int64),
		eventsHandled:   make(map[EventType]int64),
	}
}

// SubscriberCount returns the number of subscribers for an event type
func (bus *EventBus) SubscriberCount(eventType EventType) int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	return len(bus.handlers[eventType])
}
