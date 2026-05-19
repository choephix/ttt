package ui

import (
	"macro/internal/term"
	"path/filepath"
)

type Tab struct {
	Name   string
	Dirty  bool
	Active bool
}

type TabBarWidget struct {
	BaseWidget
	Tabs        []Tab
	Borders     *term.BorderSet
	ScrollOffset int
}

func NewTabBarWidget() *TabBarWidget {
	return &TabBarWidget{}
}

func (t *TabBarWidget) SetTabs(tabs []Tab) {
	t.Tabs = tabs
}

func (t *TabBarWidget) tabLabel(tab Tab) string {
	name := filepath.Base(tab.Name)
	label := " " + name
	if tab.Dirty {
		label += "*"
	}
	label += " "
	return label
}

func (t *TabBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	surface.Fill(term.Cell{Ch: ' ', Style: term.StyleInactiveTab})

	// Compute tab positions and ensure active tab is visible
	positions := make([]int, len(t.Tabs))
	total := 0
	activeStart := 0
	activeEnd := 0
	for i, tab := range t.Tabs {
		positions[i] = total
		labelW := len([]rune(t.tabLabel(tab))) + 1
		if tab.Active {
			activeStart = total
			activeEnd = total + labelW
		}
		total += labelW
	}

	if activeEnd-t.ScrollOffset > w {
		t.ScrollOffset = activeEnd - w
	}
	if activeStart < t.ScrollOffset {
		t.ScrollOffset = activeStart
	}
	if t.ScrollOffset < 0 {
		t.ScrollOffset = 0
	}

	x := -t.ScrollOffset
	for _, tab := range t.Tabs {
		label := t.tabLabel(tab)

		style := term.StyleInactiveTab
		if tab.Active {
			style = term.StyleActiveTab
		}

		for _, ch := range label {
			if x >= 0 && x < w {
				surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
			}
			x++
		}

		if x >= 0 && x < w {
			sep := '│'
			if t.Borders != nil {
				sep = t.Borders.Vertical
			}
			surface.SetCell(x, 0, term.Cell{Ch: sep, Style: term.StyleBorder})
		}
		x++
	}
}
