package ui

import (
	"macro/internal/term"
	"testing"

	"github.com/gdamore/tcell/v2"
)

type mockWidget struct {
	BaseWidget
	lastEvent    tcell.Event
	eventCount   int
	rendered     bool
	focusable    bool
	renderChar   rune
}

func (m *mockWidget) HandleEvent(ev tcell.Event) EventResult {
	m.lastEvent = ev
	m.eventCount++
	return EventConsumed
}

func (m *mockWidget) Render(surface *RenderSurface) {
	m.rendered = true
	if m.renderChar != 0 {
		surface.Fill(term.Cell{Ch: m.renderChar})
	}
}

func (m *mockWidget) Focusable() bool { return m.focusable }

func makeKeyEvent(key tcell.Key, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(key, 0, mod)
}

func TestRootRoutesToFocused(t *testing.T) {
	main := &mockWidget{}
	focused := &mockWidget{focusable: true}

	root := NewRoot(main)
	root.SetFocus(focused)
	root.SetSize(80, 24)

	ev := makeKeyEvent(tcell.KeyUp, 0)
	root.HandleEvent(ev)

	if focused.eventCount != 1 {
		t.Fatal("event not routed to focused widget")
	}
	if main.eventCount != 0 {
		t.Fatal("event incorrectly routed to main widget")
	}
}

func TestRootModalOverlayCaptures(t *testing.T) {
	main := &mockWidget{}
	focused := &mockWidget{focusable: true}
	modal := &mockWidget{focusable: true}

	root := NewRoot(main)
	root.SetFocus(focused)
	root.PushOverlay(Overlay{Widget: modal, Modal: true})

	ev := makeKeyEvent(tcell.KeyUp, 0)
	root.HandleEvent(ev)

	if modal.eventCount != 1 {
		t.Fatal("modal overlay did not receive event")
	}
	if focused.eventCount != 0 {
		t.Fatal("focused widget received event despite modal overlay")
	}
}

func TestRootGlobalKeysFire(t *testing.T) {
	main := &mockWidget{}
	focused := &mockWidget{focusable: true}
	fired := false

	root := NewRoot(main)
	root.SetFocus(focused)
	root.AddGlobalKey(tcell.KeyCtrlB, tcell.ModCtrl, 0, func() { fired = true })

	ev := makeKeyEvent(tcell.KeyCtrlB, tcell.ModCtrl)
	root.HandleEvent(ev)

	if !fired {
		t.Fatal("global key handler did not fire")
	}
	if focused.eventCount != 0 {
		t.Fatal("focused widget received event that global key should have consumed")
	}
}

func TestRootOverlayRendersOnTop(t *testing.T) {
	main := &mockWidget{renderChar: 'M'}
	overlay := &mockWidget{renderChar: 'O'}

	root := NewRoot(main)
	root.SetSize(10, 5)
	root.PushOverlay(Overlay{Widget: overlay, Modal: true})

	grid := makeGrid(10, 5)
	root.Render(grid)

	// Overlay renders last, so its character should be on top
	if grid[0][0].Ch != 'O' {
		t.Fatalf("expected overlay char 'O', got '%c'", grid[0][0].Ch)
	}
}

func TestRootPushPopOverlay(t *testing.T) {
	root := NewRoot(&mockWidget{})
	root.SetSize(80, 24)

	root.PushOverlay(Overlay{Widget: &mockWidget{}, Modal: true})
	if len(root.Overlays) != 1 {
		t.Fatal("expected 1 overlay")
	}

	root.PopOverlay()
	if len(root.Overlays) != 0 {
		t.Fatal("expected 0 overlays after pop")
	}

	// Pop on empty should not panic
	root.PopOverlay()
}
