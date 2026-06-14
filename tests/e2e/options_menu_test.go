package e2e

import (
	"testing"
)

func TestOptionsMenuToggleLineNumbers(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if !h.app.Settings.Editor.LineNumbers {
		t.Fatal("line numbers should be enabled by default")
	}

	h.exec("options.toggleLineNumbers")

	if h.app.Settings.Editor.LineNumbers {
		t.Error("line numbers should be disabled after toggle")
	}
	if h.app.EditorGroup.LineNumbers {
		t.Error("editor group line numbers should be disabled after toggle")
	}
	if h.app.EditorGroup.Editor.LineNumbers {
		t.Error("editor pane line numbers should be disabled after toggle")
	}

	h.exec("options.toggleLineNumbers")

	if !h.app.Settings.Editor.LineNumbers {
		t.Error("line numbers should be re-enabled after second toggle")
	}
	if !h.app.EditorGroup.LineNumbers {
		t.Error("editor group line numbers should be re-enabled after second toggle")
	}
	if !h.app.EditorGroup.Editor.LineNumbers {
		t.Error("editor pane line numbers should be re-enabled after second toggle")
	}
}

func TestOptionsMenuToggleWordWrap(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if h.app.Settings.Editor.WordWrap {
		t.Fatal("word wrap should be disabled by default")
	}

	h.exec("options.toggleWordWrap")

	if !h.app.Settings.Editor.WordWrap {
		t.Error("word wrap should be enabled after toggle")
	}

	h.exec("options.toggleWordWrap")

	if h.app.Settings.Editor.WordWrap {
		t.Error("word wrap should be disabled after second toggle")
	}
}

func TestOptionsMenuSetGutterStyle(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if h.app.Settings.Editor.GutterStyle != "compact" {
		t.Fatalf("expected default gutter style 'compact', got %q", h.app.Settings.Editor.GutterStyle)
	}

	h.app.SetGutterStyle("minimal")
	h.redraw()

	if h.app.Settings.Editor.GutterStyle != "minimal" {
		t.Errorf("expected gutter style 'minimal', got %q", h.app.Settings.Editor.GutterStyle)
	}
	if h.app.EditorGroup.GutterStyle != "minimal" {
		t.Errorf("expected editor group gutter style 'minimal', got %q", h.app.EditorGroup.GutterStyle)
	}
	if h.app.EditorGroup.Editor.GutterStyle != "minimal" {
		t.Errorf("expected editor pane gutter style 'minimal', got %q", h.app.EditorGroup.Editor.GutterStyle)
	}

	h.app.SetGutterStyle("extended")
	h.redraw()

	if h.app.Settings.Editor.GutterStyle != "extended" {
		t.Errorf("expected gutter style 'extended', got %q", h.app.Settings.Editor.GutterStyle)
	}
}

func TestOptionsMenuSetTabSize(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	if h.app.Settings.Editor.TabSize != 4 {
		t.Fatalf("expected default tab size 4, got %d", h.app.Settings.Editor.TabSize)
	}

	h.app.Settings.Editor.TabSize = 2
	h.app.EditorGroup.TabSize = 2
	h.app.EditorGroup.SetTabSize(2)
	h.redraw()

	if h.app.Settings.Editor.TabSize != 2 {
		t.Errorf("expected tab size 2, got %d", h.app.Settings.Editor.TabSize)
	}
	if h.app.EditorGroup.TabSize != 2 {
		t.Errorf("expected editor group tab size 2, got %d", h.app.EditorGroup.TabSize)
	}

	h.app.Settings.Editor.TabSize = 8
	h.app.EditorGroup.TabSize = 8
	h.app.EditorGroup.SetTabSize(8)
	h.redraw()

	if h.app.Settings.Editor.TabSize != 8 {
		t.Errorf("expected tab size 8, got %d", h.app.Settings.Editor.TabSize)
	}
}

func TestOptionsMenuBarPresent(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.assertContains("Options")
}

func TestOptionsMenuDynamicChecked(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	// Build the options menu and verify line numbers is checked
	items := h.app.BuildOptionsMenu()
	found := false
	for _, item := range items {
		if item.Command == "options.toggleLineNumbers" {
			found = true
			if item.Checked != 2 { // MenuChecked
				t.Errorf("expected line numbers checked (2), got %d", item.Checked)
			}
		}
	}
	if !found {
		t.Error("expected to find options.toggleLineNumbers in menu items")
	}

	// Toggle line numbers off
	h.exec("options.toggleLineNumbers")

	// Rebuild and verify unchecked
	items = h.app.BuildOptionsMenu()
	for _, item := range items {
		if item.Command == "options.toggleLineNumbers" {
			if item.Checked != 1 { // MenuUnchecked
				t.Errorf("expected line numbers unchecked (1), got %d", item.Checked)
			}
		}
	}
}
