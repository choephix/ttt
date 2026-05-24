package ui

import "ttt/internal/term"

type PanelTabBarWidget struct {
	BaseWidget
	Tabs    []Tab
	Borders *term.BorderSet
}

func NewPanelTabBarWidget() *PanelTabBarWidget {
	return &PanelTabBarWidget{}
}

func (p *PanelTabBarWidget) SetTabs(tabs []Tab) {
	p.Tabs = tabs
}

func (p *PanelTabBarWidget) Focusable() bool { return false }

func (p *PanelTabBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()

	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' '})
	}

	x := 0
	for _, tab := range p.Tabs {
		label := " " + tab.Name + " "
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
	}

}
