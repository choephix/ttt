package ui

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

func TestConfirmDialog2ButtonDefaults(t *testing.T) {
	d := NewConfirmDialogWidget("Delete?")
	if len(d.Buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(d.Buttons))
	}
	if d.Selected != 1 {
		t.Fatalf("expected default selected=1 (No), got %d", d.Selected)
	}
}

func TestConfirmDialog3ButtonDefaults(t *testing.T) {
	d := NewConfirmDialogWidget3("Save?", "Discard", "Cancel", "Save")
	if len(d.Buttons) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(d.Buttons))
	}
	if d.Selected != 2 {
		t.Fatalf("expected default selected=2, got %d", d.Selected)
	}
}

func TestConfirmDialogKeyboardNav(t *testing.T) {
	d := NewConfirmDialogWidget3("msg", "A", "B", "C")

	d.HandleEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	if d.Selected != 1 {
		t.Fatalf("left from 2: expected 1, got %d", d.Selected)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if d.Selected != 2 {
		t.Fatalf("right from 1: expected 2, got %d", d.Selected)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	if d.Selected != 0 {
		t.Fatalf("right wraps: expected 0, got %d", d.Selected)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	if d.Selected != 1 {
		t.Fatalf("tab from 0: expected 1, got %d", d.Selected)
	}
}

func TestConfirmDialogEnter(t *testing.T) {
	d := NewConfirmDialogWidget3("msg", "A", "B", "C")
	pressed := -1
	d.OnButton[0] = func() { pressed = 0 }
	d.OnButton[1] = func() { pressed = 1 }
	d.OnButton[2] = func() { pressed = 2 }

	d.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if pressed != 2 {
		t.Fatalf("enter on default(2): expected 2, got %d", pressed)
	}

	d.Selected = 0
	d.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if pressed != 0 {
		t.Fatalf("enter on 0: expected 0, got %d", pressed)
	}
}

func TestConfirmDialogEscape(t *testing.T) {
	d := NewConfirmDialogWidget("msg")
	dismissed := false
	d.OnDismiss = func() { dismissed = true }

	d.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if !dismissed {
		t.Fatal("escape should call OnDismiss")
	}
}

func TestConfirmDialogLetterShortcut(t *testing.T) {
	d := NewConfirmDialogWidget3("msg", "Discard", "Cancel", "Save")
	pressed := -1
	d.OnButton[0] = func() { pressed = 0 }
	d.OnButton[1] = func() { pressed = 1 }
	d.OnButton[2] = func() { pressed = 2 }

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	if pressed != 2 {
		t.Fatalf("'s' shortcut: expected 2 (Save), got %d", pressed)
	}

	d.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'D', tcell.ModNone))
	if pressed != 0 {
		t.Fatalf("'D' shortcut: expected 0 (Discard), got %d", pressed)
	}
}

func TestConfirmDialogUnicodeShortcut(t *testing.T) {
	// Ensure unicode buttons work for hotkey matching (not just ASCII)
	d := NewConfirmDialogWidget2("msg", "Öffnen", "Nein") // O-umlaut button
	pressed := -1
	d.OnButton[0] = func() { pressed = 0 }
	d.OnButton[1] = func() { pressed = 1 }

	// lowercase o-umlaut should match uppercase O-umlaut button
	d.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'ö', tcell.ModNone))
	if pressed != 0 {
		t.Fatalf("unicode shortcut: expected 0, got %d", pressed)
	}
}

func makeSurface(w, h int) *RenderSurface {
	cells := make([][]term.Cell, h)
	for i := range cells {
		cells[i] = make([]term.Cell, w)
	}
	return NewRenderSurface(cells, Rect{X: 0, Y: 0, W: w, H: h})
}

func TestConfirmDialogAutoSize(t *testing.T) {
	// Short message: box should fit the message width (message + 4 padding)
	d := NewConfirmDialogWidget("OK?")
	surface := makeSurface(80, 24)
	d.Render(surface)
	// "OK?" is 3 runes + 4 padding = 7
	// Buttons "Yes" + "No" = 4 + (3+4) + (2+4) = 4+7+6 = 17
	// Box should be max(7, 17) = 17
	if len(d.btnHits) != 2 {
		t.Fatalf("expected 2 hit regions, got %d", len(d.btnHits))
	}

	// Long message should make the box wider
	d2 := NewConfirmDialogWidget("This is a very long message that exceeds the button width easily")
	d2.Render(surface)
	msgW := len([]rune(d2.Message)) + 4
	if len(d2.btnHits) != 2 {
		t.Fatalf("expected 2 hit regions, got %d", len(d2.btnHits))
	}
	// The box should be at least as wide as the message
	// We verify the buttons fit within the rendered area
	for _, hit := range d2.btnHits {
		if hit.W <= 0 {
			t.Fatal("button hit region has zero or negative width")
		}
	}
	// msgW should drive the box width (it's wider than buttons)
	btnW := 4
	for _, btn := range d2.Buttons {
		btnW += len([]rune(btn)) + 4
	}
	if msgW <= btnW {
		t.Fatal("test setup: message should be wider than buttons")
	}
}

func TestConfirmDialogAutoSizeClamp(t *testing.T) {
	// On a narrow screen, box should clamp to screen width - 4
	d := NewConfirmDialogWidget("This message is quite long for a narrow terminal")
	surface := makeSurface(20, 24)
	d.Render(surface)
	// Box should be clamped to 20-4 = 16
	// Verify hit regions are within bounds
	for _, hit := range d.btnHits {
		if hit.X < 0 || hit.X+hit.W > 20 {
			t.Fatalf("button hit region out of bounds: X=%d, W=%d", hit.X, hit.W)
		}
	}
}
