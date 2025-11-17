package events

import (
	"sync"
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus returned nil")
	}
	if bus.subscribers == nil {
		t.Error("subscribers map not initialized")
	}
	if bus.closed {
		t.Error("new bus should not be closed")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	// Subscribe to event
	ch := bus.Subscribe(ServerStateChanged)

	// Publish event
	event := Event{
		Type:       ServerStateChanged,
		ServerName: "test-server",
		OldState:   "disconnected",
		NewState:   "connected",
	}

	bus.Publish(event)

	// Receive event
	select {
	case received := <-ch:
		if received.Type != ServerStateChanged {
			t.Errorf("expected type %s, got %s", ServerStateChanged, received.Type)
		}
		if received.ServerName != "test-server" {
			t.Errorf("expected server name 'test-server', got '%s'", received.ServerName)
		}
		if received.Timestamp.IsZero() {
			t.Error("timestamp should be set automatically")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	// Create multiple subscribers
	ch1 := bus.Subscribe(ServerStateChanged)
	ch2 := bus.Subscribe(ServerStateChanged)
	ch3 := bus.Subscribe(ServerStateChanged)

	// Publish event
	event := Event{
		Type:       ServerStateChanged,
		ServerName: "test-server",
	}

	bus.Publish(event)

	// All subscribers should receive the event
	for i, ch := range []<-chan Event{ch1, ch2, ch3} {
		select {
		case received := <-ch:
			if received.Type != ServerStateChanged {
				t.Errorf("subscriber %d: expected type %s, got %s", i, ServerStateChanged, received.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestNonBlockingPublish(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	// Subscribe with a channel that won't be read
	_ = bus.Subscribe(ServerStateChanged)

	// Publish more events than the buffer can hold
	// This should not block
	done := make(chan bool)
	go func() {
		for i := 0; i < 200; i++ {
			bus.Publish(Event{
				Type:       ServerStateChanged,
				ServerName: "test-server",
			})
		}
		done <- true
	}()

	// Should complete quickly without blocking
	select {
	case <-done:
		// Success - publishing didn't block
	case <-time.After(1 * time.Second):
		t.Fatal("publishing blocked even though it should be non-blocking")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	ch := bus.Subscribe(ServerStateChanged)

	// Verify subscription exists
	if count := bus.SubscriberCount(ServerStateChanged); count != 1 {
		t.Errorf("expected 1 subscriber, got %d", count)
	}

	// Unsubscribe
	bus.Unsubscribe(ServerStateChanged, ch)

	// Verify subscription removed
	if count := bus.SubscriberCount(ServerStateChanged); count != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", count)
	}

	// Publishing should not send to unsubscribed channel
	bus.Publish(Event{Type: ServerStateChanged})

	select {
	case <-ch:
		t.Error("received event after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event should be received
	}
}

func TestConcurrentPublishSubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	const numGoroutines = 10
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // publishers + subscribers

	// Start publishers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				bus.Publish(Event{
					Type: ServerStateChanged,
					Data: map[string]interface{}{
						"publisher": id,
						"seq":       j,
					},
				})
			}
		}(i)
	}

	// Start subscribers
	received := make([]int, numGoroutines)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ch := bus.Subscribe(ServerStateChanged)
			timeout := time.After(2 * time.Second)

			for {
				select {
				case <-ch:
					mu.Lock()
					received[id]++
					mu.Unlock()
				case <-timeout:
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()

	// Verify events were received (at least some, due to buffer limits and non-blocking)
	mu.Lock()
	defer mu.Unlock()

	totalReceived := 0
	for _, count := range received {
		totalReceived += count
	}

	if totalReceived == 0 {
		t.Error("no events received by any subscriber")
	}

	t.Logf("Total events published: %d, total events received: %d",
		numGoroutines*eventsPerGoroutine, totalReceived)
}

func TestClose(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe(ServerStateChanged)

	// Close bus
	bus.Close()

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after bus.Close()")
	}

	// Should be marked as closed
	if !bus.IsClosed() {
		t.Error("IsClosed() should return true after Close()")
	}

	// Publishing after close should not panic
	bus.Publish(Event{Type: ServerStateChanged})

	// Subscribing after close should return closed channel
	ch2 := bus.Subscribe(AppStateChanged)
	_, ok = <-ch2
	if ok {
		t.Error("subscribing after close should return closed channel")
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	if count := bus.SubscriberCount(ServerStateChanged); count != 0 {
		t.Errorf("expected 0 initial subscribers, got %d", count)
	}

	_ = bus.Subscribe(ServerStateChanged)
	if count := bus.SubscriberCount(ServerStateChanged); count != 1 {
		t.Errorf("expected 1 subscriber, got %d", count)
	}

	_ = bus.Subscribe(ServerStateChanged)
	if count := bus.SubscriberCount(ServerStateChanged); count != 2 {
		t.Errorf("expected 2 subscribers, got %d", count)
	}

	_ = bus.Subscribe(AppStateChanged)
	if count := bus.SubscriberCount(ServerStateChanged); count != 2 {
		t.Errorf("expected ServerStateChanged to still have 2 subscribers, got %d", count)
	}
	if count := bus.SubscriberCount(AppStateChanged); count != 1 {
		t.Errorf("expected AppStateChanged to have 1 subscriber, got %d", count)
	}
}

func TestTotalSubscribers(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	if total := bus.TotalSubscribers(); total != 0 {
		t.Errorf("expected 0 total subscribers, got %d", total)
	}

	_ = bus.Subscribe(ServerStateChanged)
	_ = bus.Subscribe(ServerStateChanged)
	_ = bus.Subscribe(AppStateChanged)

	if total := bus.TotalSubscribers(); total != 3 {
		t.Errorf("expected 3 total subscribers, got %d", total)
	}
}

func TestEventTimestamp(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	ch := bus.Subscribe(ServerStateChanged)

	// Publish event without timestamp
	event := Event{
		Type: ServerStateChanged,
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.Timestamp.IsZero() {
			t.Error("timestamp should be set automatically")
		}
		if time.Since(received.Timestamp) > time.Second {
			t.Error("timestamp should be recent")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestPublishBlocking(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	ch := bus.Subscribe(ServerStateChanged)

	// Publish blocking event
	done := make(chan bool)
	go func() {
		bus.PublishBlocking(Event{Type: ServerStateChanged})
		done <- true
	}()

	// Read the event
	select {
	case <-ch:
		// Event received
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}

	// Publishing should complete
	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("PublishBlocking did not complete")
	}
}

func TestEventData(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	ch := bus.Subscribe(ServerAutoDisabled)

	event := Event{
		Type:       ServerAutoDisabled,
		ServerName: "test-server",
		Data: map[string]interface{}{
			"reason":   "Connection failed 3 times",
			"failures": 3,
		},
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		data, ok := received.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be map[string]interface{}, got %T", received.Data)
		}
		if data["reason"] != "Connection failed 3 times" {
			t.Errorf("expected reason in data, got %v", received.Data)
		}
		if data["failures"] != 3 {
			t.Errorf("expected failures=3 in data, got %v", data["failures"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func BenchmarkPublish(b *testing.B) {
	bus := NewBus()
	defer bus.Close()

	// Single subscriber
	_ = bus.Subscribe(ServerStateChanged)

	event := Event{
		Type:       ServerStateChanged,
		ServerName: "benchmark-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}

func BenchmarkPublishMultipleSubscribers(b *testing.B) {
	bus := NewBus()
	defer bus.Close()

	// Multiple subscribers
	for i := 0; i < 10; i++ {
		_ = bus.Subscribe(ServerStateChanged)
	}

	event := Event{
		Type:       ServerStateChanged,
		ServerName: "benchmark-server",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}

func BenchmarkConcurrentPublish(b *testing.B) {
	bus := NewBus()
	defer bus.Close()

	_ = bus.Subscribe(ServerStateChanged)

	event := Event{
		Type:       ServerStateChanged,
		ServerName: "benchmark-server",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Publish(event)
		}
	})
}
