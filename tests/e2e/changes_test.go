package e2e

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestChangesKeyNavigation(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.Changes")

	if h.app.Changes.TotalChanges() == 0 {
		t.Skip("no changed files in working directory")
	}

	h.app.Changes.Selected = 0
	h.pressKey(tcell.KeyDown, tcell.ModNone)
	h.pressKey(tcell.KeyUp, tcell.ModNone)
	if h.app.Changes.Selected != 0 {
		t.Errorf("expected Selected 0 after Up, got %d", h.app.Changes.Selected)
	}
}

func TestChangesRefreshKey(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.Changes")

	h.pressRune('r')
}
