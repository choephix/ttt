package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpperCase(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "upper.txt")
	os.WriteFile(f, []byte("Hello World\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.exec("editor.selectAll")
	h.exec("editor.upperCase")
	h.redraw()

	got := h.app.EditorGroup.Editor.Buf.Lines[0]
	if got != "HELLO WORLD" {
		t.Errorf("expected 'HELLO WORLD', got %q", got)
	}
}

func TestLowerCase(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "lower.txt")
	os.WriteFile(f, []byte("Hello World\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.exec("editor.selectAll")
	h.exec("editor.lowerCase")
	h.redraw()

	got := h.app.EditorGroup.Editor.Buf.Lines[0]
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestTitleCase(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "title.txt")
	os.WriteFile(f, []byte("hello world\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.exec("editor.selectAll")
	h.exec("editor.titleCase")
	h.redraw()

	got := h.app.EditorGroup.Editor.Buf.Lines[0]
	if got != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", got)
	}
}

func TestCaseTransformNoSelection(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "nosel.txt")
	os.WriteFile(f, []byte("Hello World\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	// Run transforms without selection - should be no-op
	h.exec("editor.upperCase")
	h.exec("editor.lowerCase")
	h.exec("editor.titleCase")
	h.redraw()

	got := h.app.EditorGroup.Editor.Buf.Lines[0]
	if got != "Hello World" {
		t.Errorf("expected 'Hello World' (unchanged), got %q", got)
	}
}

func TestUpperCaseMultiLine(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	f := filepath.Join(h.dir, "multiupper.txt")
	os.WriteFile(f, []byte("hello\nworld\n"), 0644)
	h.app.EditorGroup.OpenFile(f)
	h.redraw()

	h.exec("editor.selectAll")
	h.exec("editor.upperCase")
	h.redraw()

	if h.app.EditorGroup.Editor.Buf.Lines[0] != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", h.app.EditorGroup.Editor.Buf.Lines[0])
	}
	if h.app.EditorGroup.Editor.Buf.Lines[1] != "WORLD" {
		t.Errorf("expected 'WORLD', got %q", h.app.EditorGroup.Editor.Buf.Lines[1])
	}
}
