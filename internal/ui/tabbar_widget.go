package ui

import (
	"macro/internal/term"
)

type Tab struct {
	Name   string
	Dirty  bool
	Active bool
}

type TabBarWidget struct {
	BaseWidget
	Tabs []Tab
}

func NewTabBarWidget() *TabBarWidget {
	return &TabBarWidget{}
}

func (t *TabBarWidget) SetTabs(tabs []Tab) {
	t.Tabs = tabs
}

func (t *TabBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	surface.Fill(term.Cell{Ch: ' ', Style: term.StyleInactiveTab})

	x := 0
	for _, tab := range t.Tabs {
		label := " " + tab.Name
		if tab.Dirty {
			label += "*"
		}
		label += " "

		style := term.StyleInactiveTab
		if tab.Active {
			style = term.StyleActiveTab
		}

		for _, ch := range label {
			if x >= w {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
			x++
		}

		if x < w {
			surface.SetCell(x, 0, term.Cell{Ch: '│', Style: term.StyleInactiveTab})
			x++
		}
	}
}
