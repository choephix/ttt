package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type PanelTabBarWidget struct {
	BaseWidget
	Tabs       []Tab
	Borders    *term.BorderSet
	OnTabClick func(index int)
	OnAdd      func()
	OnOverflow func(screenX, screenY int)
	MoreButton *MoreButtonWidget
	tabSpans   [][2]int
	addSpan    [2]int
	overSpan   [2]int
	HiddenTabs []int
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
	overflowW := 3 // " › "
	tabAreaW := w - rightW

	// Measure which tabs fit
	p.HiddenTabs = p.HiddenTabs[:0]
	tabWidths := make([]int, len(p.Tabs))
	total := 0
	for i, tab := range p.Tabs {
		tw := len([]rune(tab.Name)) + 2 // " name "
		if tab.Dirty {
			tw += 2 // "● "
		}
		tabWidths[i] = tw
		total += tw
	}

	// If tabs overflow, figure out how many fit with the >> button
	hasOverflow := total > tabAreaW
	if hasOverflow {
		tabAreaW -= overflowW
	}

	// Render tabs on the left
	p.tabSpans = p.tabSpans[:0]
	x := 0
	for i, tab := range p.Tabs {
		if x+tabWidths[i] > tabAreaW && hasOverflow {
			p.HiddenTabs = append(p.HiddenTabs, i)
			p.tabSpans = append(p.tabSpans, [2]int{0, 0})
			continue
		}
		style := term.StyleInactiveTab
		if tab.Active {
			style = term.StyleActiveTab
		}
		startX := x
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
		x++
		if tab.Dirty {
			surface.SetCell(x, 0, term.Cell{Ch: '●', Style: term.StyleWarning})
			x++
			surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
			x++
		}
		for _, ch := range tab.Name {
			if x >= tabAreaW {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
			x++
		}
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
		x++
		p.tabSpans = append(p.tabSpans, [2]int{startX, x})
	}

	// Render >> overflow button
	p.overSpan = [2]int{0, 0}
	if hasOverflow {
		ox := tabAreaW
		p.overSpan = [2]int{ox, ox + overflowW}
		style := term.StyleInactiveTab
		surface.SetCell(ox, 0, term.Cell{Ch: ' ', Style: style})
		surface.SetCell(ox+1, 0, term.Cell{Ch: '»', Style: style})
		surface.SetCell(ox+2, 0, term.Cell{Ch: ' ', Style: style})
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

	if p.OnOverflow != nil && p.overSpan[1] > 0 && lx >= p.overSpan[0] && lx < p.overSpan[1] {
		mx, my := mev.Position()
		p.OnOverflow(mx, my)
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
