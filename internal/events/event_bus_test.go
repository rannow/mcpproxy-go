package events

import (
	"sync"
	"testing"
	"time"

	"mcpproxy-go/internal/upstream/types"
)

func TestEventBus_PublishAndSubscribe(t *testing.T) {
	bus := NewEventBus()

	// Create a channel to receive events
	received := make(chan Event, 1)

	// Subscribe to state change events
	bus.Subscribe(EventStateChange, func(event Event) {
		received <- event
	})

	// Publish an event
	testEvent := Event{
		Type:       EventStateChange,
		ServerName: "test-server",
		Data: StateChangeData{
			OldState: types.StateDisconnected,
			NewState: types.StateReady,
		},
	}

	bus.Publish(testEvent)

	// Wait for event to be received (with timeout)
	select {
	case event := <-received:
		if event.ServerName != "test-server" {
			t.Errorf("Expected server name 'test-server', got '%s'", event.ServerName)
		}
		if event.Type != EventStateChange {
			t.Errorf("Expected event type StateChange, got %v", event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()

	// Create multiple subscribers
	const numSubscribers = 5
	var wg sync.WaitGroup
	wg.Add(numSubscribers)

	receivedCount := 0
	var mu sync.Mutex

	for i := 0; i < numSubscribers; i++ {
		bus.Subscribe(EventStateChange, func(event Event) {
			mu.Lock()
			receivedCount++
			mu.Unlock()
			wg.Done()
		})
	}

	// Publish one event
	bus.Publish(Event{
		Type:       EventStateChange,
		ServerName: "test-server",
	})

	// Wait for all subscribers to receive the event
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		mu.Lock()
		if receivedCount != numSubscribers {
			t.Errorf("Expected %d subscribers to receive event, got %d", numSubscribers, receivedCount)
		}
		mu.Unlock()
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for all subscribers")
	}
}

func TestEventBus_DifferentEventTypes(t *testing.T) {
	bus := NewEventBus()

	stateChangeReceived := make(chan bool, 1)
	configChangeReceived := make(chan bool, 1)

	// Subscribe to different event types
	bus.Subscribe(EventStateChange, func(event Event) {
		stateChangeReceived <- true
	})

	bus.Subscribe(EventConfigChange, func(event Event) {
		configChangeReceived <- true
	})

	// Publish state change event
	bus.Publish(Event{
		Type:       EventStateChange,
		ServerName: "test-server",
	})

	// Should only receive state change
	select {
	case <-stateChangeReceived:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("StateChange event not received")
	}

	select {
	case <-configChangeReceived:
		t.Error("ConfigChange handler should not have been called")
	case <-time.After(100 * time.Millisecond):
		// Expected - no config change event
	}

	// Publish config change event
	bus.Publish(Event{
		Type:       EventConfigChange,
		ServerName: "test-server",
	})

	// Should only receive config change
	select {
	case <-configChangeReceived:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("ConfigChange event not received")
	}
}

func TestEventBus_Metrics(t *testing.T) {
	bus := NewEventBus()

	// Subscribe to events
	bus.Subscribe(EventStateChange, func(event Event) {
		// Do nothing
	})

	// Publish multiple events
	for i := 0; i < 5; i++ {
		bus.Publish(Event{
			Type:       EventStateChange,
			ServerName: "test-server",
		})
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	metrics := bus.GetMetrics()

	published := metrics["published"].(map[string]int64)
	if published["StateChange"] != 5 {
		t.Errorf("Expected 5 published StateChange events, got %d", published["StateChange"])
	}

	handled := metrics["handled"].(map[string]int64)
	if handled["StateChange"] != 5 {
		t.Errorf("Expected 5 handled StateChange events, got %d", handled["StateChange"])
	}
}

func TestEventBus_SubscriberCount(t *testing.T) {
	bus := NewEventBus()

	if bus.SubscriberCount(EventStateChange) != 0 {
		t.Error("Expected 0 subscribers initially")
	}

	bus.Subscribe(EventStateChange, func(event Event) {})
	bus.Subscribe(EventStateChange, func(event Event) {})

	if bus.SubscriberCount(EventStateChange) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", bus.SubscriberCount(EventStateChange))
	}

	if bus.SubscriberCount(EventConfigChange) != 0 {
		t.Errorf("Expected 0 ConfigChange subscribers, got %d", bus.SubscriberCount(EventConfigChange))
	}
}

func TestEventBus_Reset(t *testing.T) {
	bus := NewEventBus()

	// Add subscriber and publish events
	bus.Subscribe(EventStateChange, func(event Event) {})
	bus.Publish(Event{Type: EventStateChange})

	// Reset
	bus.Reset()

	// Check that everything is cleared
	if bus.SubscriberCount(EventStateChange) != 0 {
		t.Error("Expected 0 subscribers after reset")
	}

	metrics := bus.GetMetrics()
	published := metrics["published"].(map[string]int64)
	if len(published) != 0 {
		t.Error("Expected empty published metrics after reset")
	}
}

func TestEventBus_TimestampAutoSet(t *testing.T) {
	bus := NewEventBus()

	received := make(chan Event, 1)
	bus.Subscribe(EventStateChange, func(event Event) {
		received <- event
	})

	// Publish event without timestamp
	bus.Publish(Event{
		Type:       EventStateChange,
		ServerName: "test-server",
	})

	select {
	case event := <-received:
		if event.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set automatically")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for event")
	}
}

// Benchmark tests
func BenchmarkEventBus_Publish(b *testing.B) {
	bus := NewEventBus()
	bus.Subscribe(EventStateChange, func(event Event) {
		// Do minimal work
	})

	event := Event{
		Type:       EventStateChange,
		ServerName: "test-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}

func BenchmarkEventBus_PublishWithMultipleSubscribers(b *testing.B) {
	bus := NewEventBus()

	// Add 10 subscribers
	for i := 0; i < 10; i++ {
		bus.Subscribe(EventStateChange, func(event Event) {
			// Do minimal work
		})
	}

	event := Event{
		Type:       EventStateChange,
		ServerName: "test-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}
