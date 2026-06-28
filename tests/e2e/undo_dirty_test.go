package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUndoToClearsDirty(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "dirty.txt")
	os.WriteFile(f, []byte("hello\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.pressRune('X')
	h.redraw()

	if !h.app.EditorGroup.IsDirty() {
		t.Fatal("expected dirty after typing")
	}

	h.exec("editor.undo")
	h.redraw()

	if h.app.EditorGroup.IsDirty() {
		t.Fatal("expected clean after undo to original state")
	}
}

func TestUndoToSavePoint(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "savepoint.txt")
	os.WriteFile(f, []byte("hello\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.pressRune('A')
	h.redraw()

	h.app.EditorGroup.Save()
	h.redraw()

	if h.app.EditorGroup.IsDirty() {
		t.Fatal("expected clean after save")
	}

	h.pressRune('B')
	h.redraw()

	if !h.app.EditorGroup.IsDirty() {
		t.Fatal("expected dirty after typing post-save")
	}

	h.exec("editor.undo")
	h.redraw()

	if h.app.EditorGroup.IsDirty() {
		t.Fatal("expected clean after undo to save point")
	}
}

func TestTypeOverSelectionAtomicUndo(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "replace.txt")
	os.WriteFile(f, []byte("Hello World\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.exec("editor.selectAll")
	h.redraw()

	h.pressRune('X')
	h.redraw()

	lines := h.app.EditorGroup.Editor.Buf.Lines
	if lines[0] != "X" {
		t.Fatalf("expected 'X' after typing over selection, got %q", lines[0])
	}

	h.exec("editor.undo")
	h.redraw()

	lines = h.app.EditorGroup.Editor.Buf.Lines
	if lines[0] != "Hello World" {
		t.Fatalf("expected 'Hello World' after single undo, got %q", lines[0])
	}
}
