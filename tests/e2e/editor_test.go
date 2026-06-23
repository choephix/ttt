package e2e

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestStartup(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.assertContains("File")
	h.assertContains("Edit")
	h.assertContains("View")
	h.assertContains("Explore")
}

func TestMenuBarRendered(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	row := h.screenRow(0)
	if !strings.Contains(row, "File") {
		t.Errorf("menu bar should contain 'File', got: %s", row)
	}
	if !strings.Contains(row, "Help") {
		t.Errorf("menu bar should contain 'Help', got: %s", row)
	}
}

func TestNewFile(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("file.new")
	if !h.app.EditorGroup.IsActiveVirtual() {
		t.Error("expected new file tab to be virtual")
	}
	h.assertContains("untitled")
}

func TestCommandPaletteOpenClose(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("command.palette")
	if len(h.app.Root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(h.app.Root.Overlays))
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)
	if len(h.app.Root.Overlays) != 0 {
		t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.Root.Overlays))
	}
}

func TestCommandPaletteDoesNotStack(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.pressCtrl(tcell.KeyCtrlP)
	if len(h.app.Root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay after first Ctrl+P, got %d", len(h.app.Root.Overlays))
	}

	h.pressCtrl(tcell.KeyCtrlP)
	if len(h.app.Root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay after second Ctrl+P, got %d", len(h.app.Root.Overlays))
	}
}

func TestGoToLineDialog(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("editor.goToLine")
	if len(h.app.Root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(h.app.Root.Overlays))
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)
	if len(h.app.Root.Overlays) != 0 {
		t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.Root.Overlays))
	}
}

func TestFindDialog(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("search.find")
	if len(h.app.Root.Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(h.app.Root.Overlays))
	}

	h.pressKey(tcell.KeyEscape, tcell.ModNone)
	if len(h.app.Root.Overlays) != 0 {
		t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.Root.Overlays))
	}
}

func TestThemeSwitchDialog(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("theme.switch")
	if len(h.app.Root.Overlays) == 1 {
		h.pressKey(tcell.KeyEscape, tcell.ModNone)
		if len(h.app.Root.Overlays) != 0 {
			t.Fatalf("expected 0 overlays after Escape, got %d", len(h.app.Root.Overlays))
		}
	}
}
