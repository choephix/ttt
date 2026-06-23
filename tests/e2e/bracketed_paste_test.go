package e2e

import (
	"testing"
	"time"

	"github.com/eugenioenko/ttt/internal/app"

	"github.com/gdamore/tcell/v2"
)

func runEventLoopBriefly(h *testHarness, inject func()) {
	h.t.Helper()
	running := true

	go app.RunEventLoop(
		h.app.Screen,
		h.app.Renderer,
		h.app,
		&running,
		func(panelID string) {},
	)

	inject()

	time.Sleep(200 * time.Millisecond)

	// Verify the app isn't stuck by injecting a regular key
	h.screen.InjectKey(tcell.KeyRune, 'Z', tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	running = false
	h.screen.PostEvent(tcell.NewEventInterrupt(nil))
	time.Sleep(100 * time.Millisecond)
}

func TestBracketedPasteSimple(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()
	h.exec("file.new")

	runEventLoopBriefly(h, func() {
		h.screen.PostEvent(tcell.NewEventPaste(true))
		h.screen.InjectKey(tcell.KeyRune, 'h', tcell.ModNone)
		h.screen.InjectKey(tcell.KeyRune, 'i', tcell.ModNone)
		h.screen.PostEvent(tcell.NewEventPaste(false))
	})

	buf := h.app.EditorGroup.ActiveBuffer()
	if buf == nil {
		t.Fatal("expected active buffer")
	}
	if buf.Lines[0] != "hiZ" {
		t.Errorf("expected 'hiZ', got %q", buf.Lines[0])
	}
}

func TestBracketedPasteMultiLine(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()
	h.exec("file.new")

	runEventLoopBriefly(h, func() {
		h.screen.PostEvent(tcell.NewEventPaste(true))
		// "line1\nline2\nline3"
		for _, r := range "line1" {
			h.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		}
		h.screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
		for _, r := range "line2" {
			h.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		}
		h.screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
		for _, r := range "line3" {
			h.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		}
		h.screen.PostEvent(tcell.NewEventPaste(false))
	})

	buf := h.app.EditorGroup.ActiveBuffer()
	if buf == nil {
		t.Fatal("expected active buffer")
	}
	// Should have 3 lines from paste + Z appended to last line
	if len(buf.Lines) < 3 {
		t.Errorf("expected at least 3 lines, got %d: %v", len(buf.Lines), buf.Lines)
	}
	if buf.Lines[0] != "line1" {
		t.Errorf("line 0: expected 'line1', got %q", buf.Lines[0])
	}
	if buf.Lines[1] != "line2" {
		t.Errorf("line 1: expected 'line2', got %q", buf.Lines[1])
	}
}

func TestBracketedPasteKeyboardWorksAfter(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()
	h.exec("file.new")

	running := true
	go app.RunEventLoop(
		h.app.Screen,
		h.app.Renderer,
		h.app,
		&running,
		func(panelID string) {},
	)

	// Paste
	h.screen.PostEvent(tcell.NewEventPaste(true))
	h.screen.InjectKey(tcell.KeyRune, 'A', tcell.ModNone)
	h.screen.PostEvent(tcell.NewEventPaste(false))
	time.Sleep(100 * time.Millisecond)

	// Type after paste — if pasteActive is stuck, this goes to buffer
	h.screen.InjectKey(tcell.KeyRune, 'B', tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	running = false
	h.screen.PostEvent(tcell.NewEventInterrupt(nil))
	time.Sleep(100 * time.Millisecond)

	buf := h.app.EditorGroup.ActiveBuffer()
	if buf == nil {
		t.Fatal("expected active buffer")
	}
	if buf.Lines[0] != "AB" {
		t.Errorf("expected 'AB', got %q — keyboard may be stuck after paste", buf.Lines[0])
	}
}

