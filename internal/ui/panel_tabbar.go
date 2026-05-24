package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type PanelTabBarWidget struct {
	BaseWidget
	Tabs       []Tab
	Borders    *term.BorderSet
	OnTabClick func(index int)
	OnAdd      func()
	tabSpans   [][2]int
	addSpan    [2]int
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

	p.tabSpans = p.tabSpans[:0]
	x := 0
	for _, tab := range p.Tabs {
		label := " " + tab.Name + " "
		style := term.StyleInactiveTab
		if tab.Active {
			style = term.StyleActiveTab
		}
		startX := x
		for _, ch := range label {
			if x >= w {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
			x++
		}
		p.tabSpans = append(p.tabSpans, [2]int{startX, x})
	}

	if p.OnAdd != nil && x+3 < w {
		p.addSpan = [2]int{x, x + 3}
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: term.StyleInactiveTab})
		surface.SetCell(x+1, 0, term.Cell{Ch: '+', Style: term.StyleInactiveTab})
		surface.SetCell(x+2, 0, term.Cell{Ch: ' ', Style: term.StyleInactiveTab})
	} else {
		p.addSpan = [2]int{0, 0}
	}
}

func (p *PanelTabBarWidget) HandleEvent(ev tcell.Event) EventResult {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return EventIgnored
	}
	if mev.Buttons()&tcell.Button1 == 0 {
		return EventIgnored
	}
	mx, _ := mev.Position()

	if p.OnAdd != nil && mx >= p.addSpan[0] && mx < p.addSpan[1] {
		p.OnAdd()
		return EventConsumed
	}

	for i, span := range p.tabSpans {
		if mx >= span[0] && mx < span[1] {
			if p.OnTabClick != nil {
				p.OnTabClick(i)
			}
			return EventConsumed
		}
	}
	return EventIgnored
}
