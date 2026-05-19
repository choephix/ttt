package event

import "testing"

func TestSubscribeAndPublish(t *testing.T) {
	bus := NewBus()
	received := false
	bus.Subscribe(BufferChanged, func(e Event) {
		received = true
	})

	bus.Publish(Event{Type: BufferChanged})
	if !received {
		t.Fatal("subscriber did not receive event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	count := 0
	bus.Subscribe(FileSaved, func(e Event) { count++ })
	bus.Subscribe(FileSaved, func(e Event) { count++ })

	bus.Publish(Event{Type: FileSaved})
	if count != 2 {
		t.Fatalf("expected 2 calls, got %d", count)
	}
}

func TestPublishWithNoSubscribers(t *testing.T) {
	bus := NewBus()
	// Should not panic
	bus.Publish(Event{Type: CursorMoved})
}

func TestDifferentEventTypes(t *testing.T) {
	bus := NewBus()
	bufferCalled := false
	cursorCalled := false

	bus.Subscribe(BufferChanged, func(e Event) { bufferCalled = true })
	bus.Subscribe(CursorMoved, func(e Event) { cursorCalled = true })

	bus.Publish(Event{Type: BufferChanged})

	if !bufferCalled {
		t.Fatal("buffer subscriber should have been called")
	}
	if cursorCalled {
		t.Fatal("cursor subscriber should not have been called")
	}
}
