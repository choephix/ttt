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
	MoreButton *MoreButtonWidget
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
	r := p.GetRect()

	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' '})
	}

	// Reserve right side for + and ⋮ buttons
	rightW := 0
	if p.OnAdd != nil {
		rightW += 3
	}
	if p.MoreButton != nil {
		rightW += 3
	}
	tabAreaW := w - rightW

	// Render tabs on the left
	p.tabSpans = p.tabSpans[:0]
	x := 0
	for _, tab := range p.Tabs {
		style := term.StyleInactiveTab
		if tab.Active {
			style = term.StyleActiveTab
		}
		startX := x
		label := " " + tab.Name + " "
		for _, ch := range label {
			if x >= tabAreaW {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
			x++
		}
		p.tabSpans = append(p.tabSpans, [2]int{startX, x})
	}

	// Render + and ⋮ on the right
	rx := w - rightW
	if p.OnAdd != nil {
		p.addSpan = [2]int{rx, rx + 3}
		surface.SetCell(rx, 0, term.Cell{Ch: ' ', Style: term.StyleInactiveTab})
		surface.SetCell(rx+1, 0, term.Cell{Ch: '+', Style: term.StyleInactiveTab})
		surface.SetCell(rx+2, 0, term.Cell{Ch: ' ', Style: term.StyleInactiveTab})
		rx += 3
	} else {
		p.addSpan = [2]int{0, 0}
	}

	if p.MoreButton != nil {
		p.MoreButton.SetRect(Rect{X: r.X + rx, Y: r.Y, W: 3, H: 1})
		moreSurface := surface.Sub(Rect{X: rx, Y: 0, W: 3, H: 1})
		p.MoreButton.Render(moreSurface)
	}
}

func (p *PanelTabBarWidget) HandleEvent(ev tcell.Event) EventResult {
	// Check MoreButton first (it uses absolute coords internally)
	if p.MoreButton != nil {
		if result := p.MoreButton.HandleEvent(ev); result == EventConsumed {
			return EventConsumed
		}
	}

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
