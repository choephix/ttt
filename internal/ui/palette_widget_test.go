package ui

import (
	"github.com/eugenioenko/ttt/internal/command"
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

	if len(p.Items) != 4 {
		t.Fatalf("empty query: expected 4, got %d", len(p.Items))
	}

	// Initial text is "> ", typing appends after it
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'f', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'i', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'l', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0))

	if len(p.Items) != 2 {
		t.Fatalf("query 'file': expected 2 results, got %d", len(p.Items))
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

	// Wraps to last item
	p.HandleEvent(tcell.NewEventKey(tcell.KeyUp, 0, 0))
	if p.Selected != 3 {
		t.Fatalf("expected selected 3 (wrapped), got %d", p.Selected)
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

	// Initial text is ">", type "sp" → ">sp"
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 's', 0))
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'p', 0))
	if p.Input.Text != ">sp" {
		t.Fatalf("expected text '>sp', got '%s'", p.Input.Text)
	}

	p.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace2, 0, 0))
	if p.Input.Text != ">s" {
		t.Fatalf("expected text '>s' after backspace, got '%s'", p.Input.Text)
	}
}

func TestPaletteModeSwitch(t *testing.T) {
	p := NewCommandPaletteWidget(testCommands())

	if p.mode != paletteCommandMode {
		t.Fatal("expected command mode initially")
	}

	// Clear input to switch to file mode
	p.Input.Clear()
	if p.mode != paletteFileMode {
		t.Fatal("expected file mode after clearing input")
	}

	// Type ">" to switch back to command mode
	p.HandleEvent(tcell.NewEventKey(tcell.KeyRune, '>', 0))
	if p.mode != paletteCommandMode {
		t.Fatal("expected command mode after typing '>'")
	}
	if len(p.Items) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(p.Items))
	}
}
