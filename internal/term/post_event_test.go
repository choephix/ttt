package term

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
)

// tScreen.Fini closes the event queue; PostEvent must drop events instead of
// panicking when an async poster races shutdown (send on a closed channel
// panics even inside a select with a default case).
func TestPostEventAfterQueueClosedDoesNotPanic(t *testing.T) {
	sim := NewSimScreen()
	if err := sim.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	scr := NewTcellScreenFrom(sim)

	close(sim.EventQ())

	if err := scr.PostEvent(tcell.NewEventInterrupt(nil)); err != nil {
		t.Fatalf("PostEvent returned error: %v", err)
	}
}

// The queue-full fallback goroutine must also survive the queue closing while
// it is blocked on the send.
func TestPostEventFallbackSurvivesQueueClose(t *testing.T) {
	sim := NewSimScreen()
	if err := sim.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	scr := NewTcellScreenFrom(sim)

	q := sim.EventQ()
	for i := 0; i < cap(q); i++ {
		q <- tcell.NewEventInterrupt(nil)
	}
	// Queue is full: this send takes the goroutine fallback path and blocks.
	if err := scr.PostEvent(tcell.NewEventInterrupt(nil)); err != nil {
		t.Fatalf("PostEvent returned error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	close(q)
	time.Sleep(10 * time.Millisecond)
}
