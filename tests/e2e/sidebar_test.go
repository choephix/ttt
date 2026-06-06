package e2e

import (
	"strings"
	"testing"
)

func TestSidebarTabClick(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.app.SplitPanel.DividerPos = 40
	h.app.Root.SetSize(80, 24)
	h.redraw()

	if h.app.Sidebar.ActivePanel != "explorer" {
		t.Fatalf("expected active panel 'explorer', got %q", h.app.Sidebar.ActivePanel)
	}

	sidebarY := h.app.Sidebar.GetRect().Y
	sidebarX := h.app.Sidebar.GetRect().X

	h.click(sidebarX+10, sidebarY)
	if h.app.Sidebar.ActivePanel != "search" {
		t.Errorf("expected active panel 'search' after click, got %q", h.app.Sidebar.ActivePanel)
	}

	h.click(sidebarX+18, sidebarY)
	if h.app.Sidebar.ActivePanel != "changes" {
		t.Errorf("expected active panel 'changes' after click, got %q", h.app.Sidebar.ActivePanel)
	}

	h.click(sidebarX+3, sidebarY)
	if h.app.Sidebar.ActivePanel != "explorer" {
		t.Errorf("expected active panel 'explorer' after click, got %q", h.app.Sidebar.ActivePanel)
	}
}

func TestToggleSidebar(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.assertContains("Explore")

	h.exec("sidebar.toggle")
	h.assertNotContains("Explore")

	h.exec("sidebar.toggle")
	h.assertContains("Explore")
}

func TestSidebarPanelSwitching(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.Explorer")
	if h.app.Sidebar.ActivePanel != "explorer" {
		t.Errorf("expected active panel 'explorer', got %q", h.app.Sidebar.ActivePanel)
	}

	h.exec("sidebar.search")
	if h.app.Sidebar.ActivePanel != "search" {
		t.Errorf("expected active panel 'search', got %q", h.app.Sidebar.ActivePanel)
	}

	h.exec("sidebar.Changes")
	if h.app.Sidebar.ActivePanel != "changes" {
		t.Errorf("expected active panel 'changes', got %q", h.app.Sidebar.ActivePanel)
	}
}

func TestSidebarWidth(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	initial := h.app.SplitPanel.DividerPos

	h.exec("sidebar.wider")
	if h.app.SplitPanel.DividerPos != initial+1 {
		t.Errorf("expected width %d, got %d", initial+1, h.app.SplitPanel.DividerPos)
	}

	h.exec("sidebar.narrower")
	if h.app.SplitPanel.DividerPos != initial {
		t.Errorf("expected width %d, got %d", initial, h.app.SplitPanel.DividerPos)
	}
}

func TestFocusEditor(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.exec("sidebar.focus")
	if h.app.Root.Focused == h.app.EditorGroup {
		t.Error("focus should not be on editor after sidebar.focus")
	}

	h.exec("editor.focus")
	if h.app.Root.Focused != h.app.EditorGroup {
		t.Error("focus should be on editor after editor.focus")
	}
}

func TestSidebarTabOverflow(t *testing.T) {
	h := newTestHarness(t, 80, 24)
	defer h.stop()

	h.app.SplitPanel.DividerPos = 20
	h.app.Root.SetSize(80, 24)
	h.redraw()

	sidebarY := h.app.Sidebar.GetRect().Y
	row := h.screenRow(sidebarY)
	t.Logf("sidebar row (w=20): %q", row)

	if !strings.Contains(row, "»") {
		t.Errorf("expected overflow » indicator in narrow sidebar, got: %s", row)
	}

	if len(h.app.Sidebar.TabBar.HiddenTabs) == 0 {
		t.Error("expected at least one hidden tab due to overflow")
	}

	h.app.SplitPanel.DividerPos = 30
	h.app.Root.SetSize(80, 24)
	h.redraw()

	row = h.screenRow(sidebarY)
	t.Logf("sidebar row (w=30): %q", row)

	if strings.Contains(row, "»") {
		t.Errorf("expected no overflow with default sidebar, got: %s", row)
	}

	if len(h.app.Sidebar.TabBar.HiddenTabs) != 0 {
		t.Errorf("expected 0 hidden tabs with default sidebar, got %d", len(h.app.Sidebar.TabBar.HiddenTabs))
	}
}
