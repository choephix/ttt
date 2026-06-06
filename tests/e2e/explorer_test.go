package e2e

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestExplorerKeyNavigation(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	if len(h.app.Explorer.FlatList) < 3 {
		t.Skipf("expected at least 3 explorer items, got %d", len(h.app.Explorer.FlatList))
	}

	h.app.Explorer.Selected = 0

	h.pressKey(tcell.KeyDown, tcell.ModNone)
	if h.app.Explorer.Selected != 1 {
		t.Errorf("expected Selected 1 after Down, got %d", h.app.Explorer.Selected)
	}

	h.pressKey(tcell.KeyDown, tcell.ModNone)
	if h.app.Explorer.Selected != 2 {
		t.Errorf("expected Selected 2 after second Down, got %d", h.app.Explorer.Selected)
	}

	h.pressKey(tcell.KeyUp, tcell.ModNone)
	if h.app.Explorer.Selected != 1 {
		t.Errorf("expected Selected 1 after Up, got %d", h.app.Explorer.Selected)
	}

	h.app.Explorer.Selected = 0
	h.pressKey(tcell.KeyUp, tcell.ModNone)
	if h.app.Explorer.Selected != 0 {
		t.Errorf("expected Selected 0 (clamped at top), got %d", h.app.Explorer.Selected)
	}
}

func TestExplorerDirExpandCollapse(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	h.app.Explorer.Selected = 0
	root := h.app.Explorer.FlatList[0]
	if !root.IsDir {
		t.Fatal("expected root to be a directory")
	}

	initialCount := len(h.app.Explorer.FlatList)

	h.pressKey(tcell.KeyLeft, tcell.ModNone)
	if root.Expanded {
		t.Error("expected root to be collapsed after Left")
	}
	if len(h.app.Explorer.FlatList) >= initialCount {
		t.Error("expected fewer items after collapsing root")
	}

	h.pressKey(tcell.KeyRight, tcell.ModNone)
	if !root.Expanded {
		t.Error("expected root to be expanded after Right")
	}
	if len(h.app.Explorer.FlatList) != initialCount {
		t.Errorf("expected %d items after re-expanding, got %d", initialCount, len(h.app.Explorer.FlatList))
	}
}

func TestExplorerEnterOpensFile(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	fileIdx := -1
	for i, node := range h.app.Explorer.FlatList {
		if !node.IsDir {
			fileIdx = i
			break
		}
	}
	if fileIdx < 0 {
		t.Skip("no file found in explorer")
	}

	h.app.Explorer.Selected = fileIdx
	expectedPath := h.app.Explorer.FlatList[fileIdx].Path

	h.pressKey(tcell.KeyEnter, tcell.ModNone)

	if h.app.EditorGroup.ActiveFilePath() != expectedPath {
		t.Errorf("expected editor to open %q, got %q", expectedPath, h.app.EditorGroup.ActiveFilePath())
	}
}

func TestExplorerEnterToggleDir(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	h.app.Explorer.Selected = 0
	root := h.app.Explorer.FlatList[0]
	if !root.IsDir || !root.Expanded {
		t.Fatal("expected root to be an expanded directory")
	}

	h.pressKey(tcell.KeyEnter, tcell.ModNone)
	if root.Expanded {
		t.Error("expected root to be collapsed after Enter")
	}

	h.pressKey(tcell.KeyEnter, tcell.ModNone)
	if !root.Expanded {
		t.Error("expected root to be expanded after second Enter")
	}
}

func TestExplorerClickOpensFile(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")
	h.redraw()

	fileIdx := -1
	for i, node := range h.app.Explorer.FlatList {
		if !node.IsDir {
			fileIdx = i
			break
		}
	}
	if fileIdx < 0 {
		t.Skip("no file found in explorer")
	}

	r := h.app.Explorer.GetRect()
	clickY := r.Y + (fileIdx - h.app.Explorer.ScrollTop)
	h.click(r.X+5, clickY)

	if h.app.Explorer.Selected != fileIdx {
		t.Errorf("expected Selected %d after click, got %d", fileIdx, h.app.Explorer.Selected)
	}

	expectedPath := h.app.Explorer.FlatList[fileIdx].Path
	if h.app.EditorGroup.ActiveFilePath() != expectedPath {
		t.Errorf("expected editor to open %q, got %q", expectedPath, h.app.EditorGroup.ActiveFilePath())
	}
}

func TestExplorerScrollFollowing(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.explorer")

	itemCount := len(h.app.Explorer.FlatList)
	if itemCount < 5 {
		t.Skipf("need at least 5 items for scroll test, got %d", itemCount)
	}

	h.app.Explorer.Selected = itemCount - 1
	r := h.app.Explorer.GetRect()
	contentH := r.H

	h.redraw()

	if contentH > 0 && itemCount > contentH {
		if h.app.Explorer.ScrollTop == 0 {
			t.Error("expected ScrollTop > 0 when selected item is past visible area")
		}
	}
}
