package events

import (
	"sync"
	"time"

	"mcpproxy-go/internal/config"
)

// EventType represents the type of event
type EventType string

const (
	// Server state events
	ServerStateChanged  EventType = "server_state_changed"
	ServerConfigChanged EventType = "server_config_changed"
	ServerAutoDisabled  EventType = "server_auto_disabled"
	ServerGroupUpdated  EventType = "server_group_updated"

	// Application state events
	AppStateChanged EventType = "app_state_changed"

	// HIGH-006: Legacy event type names - kept for backward compatibility
	// These are actively used throughout the codebase and should be migrated
	// to the canonical names above in a future refactoring phase.
	// Canonical mappings:
	//   EventStateChange    → ServerStateChanged (for server state changes)
	//   EventAppStateChange → AppStateChanged (for app state changes)
	//   EventConfigChange   → ServerConfigChanged (for config changes)
	EventStateChange    EventType = "state_change"     // Legacy: use ServerStateChanged for new code
	EventAppStateChange EventType = "app_state_change" // Legacy: use AppStateChanged for new code
	EventConfigChange   EventType = "config_change"    // Legacy: use ServerConfigChanged for new code

	// Tool events
	ToolsUpdated EventType = "tools_updated"
	ToolCalled   EventType = "tool_called"

	// Connection events
	ConnectionEstablished EventType = "connection_established"
	ConnectionLost        EventType = "connection_lost"
)

// EventBus is an alias for Bus for backward compatibility
type EventBus = Bus

// NewEventBus creates a new event bus (alias for NewBus)
func NewEventBus() *EventBus {
	return NewBus()
}

// Data structures for specific event types

// StateChangeData contains data for state change events
type StateChangeData struct {
	ServerName string      `json:"server_name,omitempty"`
	OldState   interface{} `json:"old_state"` // Can be string or ConnectionState
	NewState   interface{} `json:"new_state"` // Can be string or ConnectionState
	Info       interface{} `json:"info,omitempty"`
}

// AppStateChangeData contains data for app state change events
type AppStateChangeData struct {
	OldState string `json:"old_state"`
	NewState string `json:"new_state"`
}

// ConfigChangeData contains data for config change events
type ConfigChangeData struct {
	Action string `json:"action"` // "created", "updated", "deleted"
}

// Event represents a single event in the system
type Event struct {
	Type       EventType   `json:"type"`
	ServerName string      `json:"server_name,omitempty"`
	OldState   string      `json:"old_state,omitempty"`
	NewState   string      `json:"new_state,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
	Data       interface{} `json:"data,omitempty"` // Can be map[string]interface{} or specific data types
}

// Bus is a thread-safe event bus for pub/sub messaging
type Bus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]chan Event
	closed      bool
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[EventType][]chan Event),
		closed:      false,
	}
}

// Subscribe subscribes to a specific event type and returns a channel for receiving events
// The channel is buffered to prevent blocking publishers
func (b *Bus) Subscribe(eventType EventType) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		// Return a closed channel if bus is closed
		ch := make(chan Event)
		close(ch)
		return ch
	}

	// Create buffered channel to prevent blocking
	ch := make(chan Event, config.EventChannelBufferSize)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

// SubscribeAll subscribes to all event types
func (b *Bus) SubscribeAll() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		ch := make(chan Event)
		close(ch)
		return ch
	}

	// Create a larger buffer for all events
	ch := make(chan Event, config.EventChannelBufferSizeAll)

	// Subscribe to all known event types
	for eventType := range b.subscribers {
		b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	}

	// Also add to future event types (this is a limitation - we need to track "all" subscribers separately)
	// For now, we'll handle this in Publish by checking for wildcard subscribers

	return ch
}

// Unsubscribe removes a subscription channel
func (b *Bus) Unsubscribe(eventType EventType, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, exists := b.subscribers[eventType]
	if !exists {
		return
	}

	// Find and remove the channel
	for i, subscriber := range subscribers {
		if subscriber == ch {
			// Remove from slice without preserving order (more efficient)
			b.subscribers[eventType][i] = b.subscribers[eventType][len(b.subscribers[eventType])-1]
			b.subscribers[eventType] = b.subscribers[eventType][:len(b.subscribers[eventType])-1]
			break
		}
	}

	// Clean up empty subscriber lists
	if len(b.subscribers[eventType]) == 0 {
		delete(b.subscribers, eventType)
	}
}

// Publish publishes an event to all subscribers of that event type
// This method is non-blocking - if a subscriber's channel is full, the event is dropped
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Send to all subscribers of this event type
	subscribers, exists := b.subscribers[event.Type]
	if exists {
		for _, ch := range subscribers {
			select {
			case ch <- event:
				// Event sent successfully
			default:
				// Channel is full, drop event to prevent blocking
				// In production, we might want to log this or track dropped events
			}
		}
	}
}

// PublishBlocking publishes an event and blocks until all subscribers have received it
// Use this sparingly as it can cause deadlocks if subscribers are slow
func (b *Bus) PublishBlocking(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Send to all subscribers (blocking)
	subscribers, exists := b.subscribers[event.Type]
	if exists {
		for _, ch := range subscribers {
			ch <- event
		}
	}
}

// Close closes the event bus and all subscriber channels
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true

	// Close all subscriber channels
	for _, subscribers := range b.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
	}

	// Clear subscribers
	b.subscribers = make(map[EventType][]chan Event)
}

// SubscriberCount returns the number of subscribers for a specific event type
func (b *Bus) SubscriberCount(eventType EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers[eventType])
}

// TotalSubscribers returns the total number of subscriber channels across all event types
func (b *Bus) TotalSubscribers() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	total := 0
	for _, subscribers := range b.subscribers {
		total += len(subscribers)
	}
	return total
}

// IsClosed returns whether the bus has been closed
func (b *Bus) IsClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.closed
}
