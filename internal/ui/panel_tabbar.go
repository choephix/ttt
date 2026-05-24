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
	OnTabClose func(index int)
	OnAdd      func()
	tabSpans   [][2]int
	closeSpans [][2]int
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
	p.closeSpans = p.closeSpans[:0]
	x := 0
	for _, tab := range p.Tabs {
		style := term.StyleInactiveTab
		if tab.Active {
			style = term.StyleActiveTab
		}
		startX := x
		label := " " + tab.Name + " "
		for _, ch := range label {
			if x >= w {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
			x++
		}
		if tab.Active && p.OnTabClose != nil && x+2 <= w {
			closeStart := x
			surface.SetCell(x, 0, term.Cell{Ch: '×', Style: style})
			x++
			surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
			x++
			p.closeSpans = append(p.closeSpans, [2]int{closeStart, x})
		} else {
			p.closeSpans = append(p.closeSpans, [2]int{0, 0})
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
	r := p.GetRect()
	lx := mx - r.X

	if p.OnAdd != nil && lx >= p.addSpan[0] && lx < p.addSpan[1] {
		p.OnAdd()
		return EventConsumed
	}

	for i, span := range p.closeSpans {
		if span[0] != span[1] && lx >= span[0] && lx < span[1] {
			if p.OnTabClose != nil {
				p.OnTabClose(i)
			}
			return EventConsumed
		}
	}

	for i, span := range p.tabSpans {
		if lx >= span[0] && lx < span[1] {
			if p.OnTabClick != nil {
				p.OnTabClick(i)
			}
			return EventConsumed
		}
	}
	return EventIgnored
}
