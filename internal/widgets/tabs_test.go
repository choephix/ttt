package widgets

import (
	"strings"
	"testing"
)

func tabsRowText(s *testSurface, y int) string {
	if y < 0 || y >= len(s.cells) {
		return ""
	}
	runes := make([]rune, len(s.cells[y]))
	for x, c := range s.cells[y] {
		if c.Ch == 0 {
			runes[x] = ' '
		} else {
			runes[x] = c.Ch
		}
	}
	return string(runes)
}

func TestTabsAllFit(t *testing.T) {
	tw := NewTabsWidget(TabsConfig{
		Items: []TabItem{
			{ID: "a", Label: "Explorer", Active: true},
			{ID: "b", Label: "Search"},
			{ID: "c", Label: "Changes"},
		},
	})
	s := renderWidget(tw, 0, 0, 40, 1)
	row := tabsRowText(s, 0)
	if !strings.Contains(row, "Explorer") || !strings.Contains(row, "Search") || !strings.Contains(row, "Changes") {
		t.Fatalf("all tabs should be visible, got: %q", row)
	}
	if len(tw.HiddenTabs()) != 0 {
		t.Fatal("no tabs should be hidden")
	}
}

func TestTabsOverflowActiveFirst(t *testing.T) {
	tw := NewTabsWidget(TabsConfig{
		Items: []TabItem{
			{ID: "a", Label: "Explorer"},
			{ID: "b", Label: "Search"},
			{ID: "c", Label: "Changes"},
			{ID: "d", Label: "Docker", Active: true},
		},
	})
	// Width too narrow for all four tabs
	s := renderWidget(tw, 0, 0, 30, 1)
	row := tabsRowText(s, 0)
	if !strings.Contains(row, "Docker") {
		t.Fatalf("active tab 'Docker' should always be visible, got: %q", row)
	}
	if !strings.Contains(row, "»") {
		t.Fatalf("should have overflow chevron, got: %q", row)
	}
}

func TestTabsOverflowPreservesOrder(t *testing.T) {
	tw := NewTabsWidget(TabsConfig{
		Items: []TabItem{
			{ID: "a", Label: "Explorer"},
			{ID: "b", Label: "Search"},
			{ID: "c", Label: "Changes", Active: true},
			{ID: "d", Label: "Docker"},
		},
	})
	s := renderWidget(tw, 0, 0, 30, 1)
	row := tabsRowText(s, 0)
	if !strings.Contains(row, "Changes") {
		t.Fatalf("active tab should be visible, got: %q", row)
	}
	// Visible tabs should maintain their original order
	explorerPos := strings.Index(row, "Explorer")
	changesPos := strings.Index(row, "Changes")
	if explorerPos >= 0 && changesPos >= 0 && explorerPos > changesPos {
		t.Fatalf("visible tabs should maintain original order, got: %q", row)
	}
}

func TestTabsOverflowOnlyActiveFits(t *testing.T) {
	tw := NewTabsWidget(TabsConfig{
		Items: []TabItem{
			{ID: "a", Label: "Explorer"},
			{ID: "b", Label: "Search"},
			{ID: "c", Label: "Docker", Active: true},
		},
	})
	// Very narrow: only active tab + chevron should fit
	s := renderWidget(tw, 0, 0, 12, 1)
	row := tabsRowText(s, 0)
	if !strings.Contains(row, "Docker") {
		t.Fatalf("active tab should be visible even when very narrow, got: %q", row)
	}
	if len(tw.HiddenTabs()) != 2 {
		t.Fatalf("expected 2 hidden tabs, got %d", len(tw.HiddenTabs()))
	}
}

func TestTabsHiddenTabsNotClickable(t *testing.T) {
	tw := NewTabsWidget(TabsConfig{
		Items: []TabItem{
			{ID: "a", Label: "Explorer"},
			{ID: "b", Label: "Search"},
			{ID: "c", Label: "Changes"},
			{ID: "d", Label: "Docker", Active: true},
		},
	})
	renderWidget(tw, 0, 0, 30, 1)
	for _, idx := range tw.HiddenTabs() {
		span := tw.tabSpans[idx]
		if span[0] != 0 || span[1] != 0 {
			t.Fatalf("hidden tab %d should have zero span, got %v", idx, span)
		}
	}
}
