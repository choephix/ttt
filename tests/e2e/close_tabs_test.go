package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCloseAllTabsNoDirty(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "clean.txt")
	os.WriteFile(f, []byte("hello\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.exec("tab.closeAll")
	h.redraw()

	name := h.app.EditorGroup.ActiveFileName()
	if name != "untitled" {
		t.Errorf("expected untitled after close all, got %q", name)
	}
}

func TestCloseAllTabsDirtyShowsDialog(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "dirty.txt")
	os.WriteFile(f, []byte("hello\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.pressRune('X')
	h.redraw()

	if !h.app.EditorGroup.IsDirty() {
		t.Fatal("expected dirty buffer")
	}

	h.exec("tab.closeAll")
	h.redraw()

	if !h.app.Root.HasOverlay() {
		t.Fatal("expected confirm dialog for dirty tabs")
	}
}

func TestCloseAllSavedKeepsDirty(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	clean := filepath.Join(h.dir, "clean.txt")
	dirty := filepath.Join(h.dir, "dirty.txt")
	os.WriteFile(clean, []byte("clean\n"), 0644)
	os.WriteFile(dirty, []byte("dirty\n"), 0644)

	h.app.EditorGroup.OpenFile(clean)
	h.app.EditorGroup.OpenFile(dirty)
	h.redraw()

	h.pressRune('X')
	h.redraw()

	h.exec("tab.closeAllSaved")
	h.redraw()

	name := h.app.EditorGroup.ActiveFileName()
	if name != "dirty.txt" {
		t.Errorf("expected dirty tab to remain, got %q", name)
	}
	if !h.app.EditorGroup.IsDirty() {
		t.Error("expected remaining tab to be dirty")
	}
}
