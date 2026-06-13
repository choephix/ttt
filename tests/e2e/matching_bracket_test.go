package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoToMatchingBracket_Forward(t *testing.T) {
	h := newTestHarness(t, 80, 30)
	defer h.stop()
	f := filepath.Join(h.dir, "test.go")
	os.WriteFile(f, []byte("func main() {\n\tx()\n}\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()
	// Position cursor at the opening brace on line 0
	h.app.EditorGroup.Editor.Cursor.Line = 0
	h.app.EditorGroup.Editor.Cursor.Col = 12 // the '{' character
	h.redraw()
	h.exec("editor.goToMatchingBracket")
	c := h.app.EditorGroup.Editor.Cursor
	if c.Line != 2 || c.Col != 0 {
		t.Errorf("expected cursor at (2,0) for closing brace, got (%d,%d)", c.Line, c.Col)
	}
}

func TestGoToMatchingBracket_Backward(t *testing.T) {
	h := newTestHarness(t, 80, 30)
	defer h.stop()
	f := filepath.Join(h.dir, "test.go")
	os.WriteFile(f, []byte("func main() {\n\tx()\n}\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()
	// Position cursor at the closing brace
	h.app.EditorGroup.Editor.Cursor.Line = 2
	h.app.EditorGroup.Editor.Cursor.Col = 0
	h.redraw()
	h.exec("editor.goToMatchingBracket")
	c := h.app.EditorGroup.Editor.Cursor
	if c.Line != 0 || c.Col != 12 {
		t.Errorf("expected cursor at (0,12) for opening brace, got (%d,%d)", c.Line, c.Col)
	}
}

func TestGoToMatchingBracket_NoMatch(t *testing.T) {
	h := newTestHarness(t, 80, 30)
	defer h.stop()
	f := filepath.Join(h.dir, "test.txt")
	os.WriteFile(f, []byte("no brackets here\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()
	h.app.EditorGroup.Editor.Cursor.Line = 0
	h.app.EditorGroup.Editor.Cursor.Col = 0
	h.redraw()
	h.exec("editor.goToMatchingBracket")
	c := h.app.EditorGroup.Editor.Cursor
	if c.Line != 0 || c.Col != 0 {
		t.Errorf("expected cursor unchanged at (0,0), got (%d,%d)", c.Line, c.Col)
	}
}
