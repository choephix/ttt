package ui

import (
	"ttt/internal/command"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func testCommands() []command.Command {
	return []command.Command{
		{ID: "file.save", Title: "File: Save"},
		{ID: "file.open", Title: "File: Open"},
		{ID: "editor.split", Title: "Split Editor Right"},
		{ID: "sidebar.toggle", Title: "Toggle Sidebar"},
	}
}

func TestPaletteFilter(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())

	if len(p.Filtered) != 4 {
		t.Fatalf("empty query: expected 4, got %d", len(p.Filtered))
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'f', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'l', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))

	if len(p.Filtered) != 2 {
		t.Fatalf("query 'file': expected 2 results, got %d", len(p.Filtered))
	}
}

func TestPaletteNavigation(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())

	p.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if p.Selected != 1 {
		t.Fatalf("expected selected 1, got %d", p.Selected)
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if p.Selected != 0 {
		t.Fatalf("expected selected 0, got %d", p.Selected)
	}

	// Can't go above 0
	p.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if p.Selected != 0 {
		t.Fatalf("expected selected 0 (clamped), got %d", p.Selected)
	}
}

func TestPaletteExecute(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())
	executed := ""
	p.OnExecute = func(id string) { executed = id }

	p.HandleEvent(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))

	if executed != "file.open" {
		t.Fatalf("expected 'file.open', got '%s'", executed)
	}
}

func TestPaletteDismiss(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())
	dismissed := false
	p.OnDismiss = func() { dismissed = true }

	p.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))

	if !dismissed {
		t.Fatal("palette should have been dismissed")
	}
}

func TestPaletteAlwaysConsumes(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())
	result := p.HandleEvent(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	if result != EventConsumed {
		t.Fatal("palette should consume all events (modal)")
	}
}

func TestPaletteBackspace(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())

	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 's', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
	if p.Query != "sp" {
		t.Fatalf("expected query 'sp', got '%s'", p.Query)
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace2, 0, 0))
	if p.Query != "s" {
		t.Fatalf("expected query 's' after backspace, got '%s'", p.Query)
	}
}
