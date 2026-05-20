package ui

import (
	"ttt/internal/term"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
)

type tabSpan struct {
	start, end int
	label      string
	active     bool
}

type Tab struct {
	Name   string
	Dirty  bool
	Active bool
}

type TabBarWidget struct {
	BaseWidget
	Tabs         []Tab
	Borders      *term.BorderSet
	ScrollOffset int
	OnTabClick   func(index int)
	tabSpans     []tabSpan
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

	b := term.SingleBorderSet()
	if t.Borders != nil {
		b = *t.Borders
	}
	bs := term.StyleBorder

	// Compute spans: active tab needs 2 extra cols for │ │ side borders
	spans := make([]tabSpan, len(t.Tabs))
	pos := 0
	activeIdx := -1
	for i, tab := range t.Tabs {
		label := t.tabLabel(tab)
		labelW := len([]rune(label))
		spanW := labelW
		if tab.Active {
			spanW += 2
		}
		spans[i] = tabSpan{start: pos, end: pos + spanW, label: label, active: tab.Active}
		if tab.Active {
			activeIdx = i
		}
		pos += spanW
	}
	t.tabSpans = spans

	// Scroll to keep active tab visible
	if activeIdx >= 0 {
		s := spans[activeIdx]
		if s.end-t.ScrollOffset > w {
			t.ScrollOffset = s.end - w
		}
		if s.start < t.ScrollOffset {
			t.ScrollOffset = s.start
		}
	}
	if t.ScrollOffset < 0 {
		t.ScrollOffset = 0
	}

	// Row 0: top of active tab ┌───┐, spaces elsewhere
	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' '})
	}
	if activeIdx >= 0 {
		s := spans[activeIdx]
		sx := s.start - t.ScrollOffset
		ex := s.end - t.ScrollOffset
		if sx >= 0 && sx < w {
			surface.SetCell(sx, 0, term.Cell{Ch: b.TopLeft, Style: bs})
		}
		for x := sx + 1; x < ex-1; x++ {
			if x >= 0 && x < w {
				surface.SetCell(x, 0, term.Cell{Ch: b.Horizontal, Style: bs})
			}
		}
		if ex-1 > sx && ex-1 >= 0 && ex-1 < w {
			surface.SetCell(ex-1, 0, term.Cell{Ch: b.TopRight, Style: bs})
		}
	}

	// Row 1: tab labels, active tab has │ sides
	for x := 0; x < w; x++ {
		surface.SetCell(x, 1, term.Cell{Ch: ' '})
	}
	for _, s := range spans {
		sx := s.start - t.ScrollOffset
		ex := s.end - t.ScrollOffset
		if s.active {
			if sx >= 0 && sx < w {
				surface.SetCell(sx, 1, term.Cell{Ch: b.Vertical, Style: bs})
			}
			for ci, ch := range []rune(s.label) {
				x := sx + 1 + ci
				if x >= 0 && x < w {
					surface.SetCell(x, 1, term.Cell{Ch: ch, Style: term.StyleActiveTab})
				}
			}
			if ex-1 >= 0 && ex-1 < w {
				surface.SetCell(ex-1, 1, term.Cell{Ch: b.Vertical, Style: bs})
			}
		} else {
			for ci, ch := range []rune(s.label) {
				x := sx + ci
				if x >= 0 && x < w {
					surface.SetCell(x, 1, term.Cell{Ch: ch, Style: term.StyleInactiveTab})
				}
			}
		}
	}

	// Row 2: baseline ─┘  └─, horizontal line with gap for active tab
	for x := 0; x < w; x++ {
		surface.SetCell(x, 2, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	if activeIdx >= 0 {
		s := spans[activeIdx]
		sx := s.start - t.ScrollOffset
		ex := s.end - t.ScrollOffset
		if sx >= 0 && sx < w {
			surface.SetCell(sx, 2, term.Cell{Ch: b.BottomRight, Style: bs})
		}
		for x := sx + 1; x < ex-1; x++ {
			if x >= 0 && x < w {
				surface.SetCell(x, 2, term.Cell{Ch: ' '})
			}
		}
		if ex-1 > sx && ex-1 >= 0 && ex-1 < w {
			surface.SetCell(ex-1, 2, term.Cell{Ch: b.BottomLeft, Style: bs})
		}
	}
}

func (t *TabBarWidget) HandleEvent(ev tcell.Event) EventResult {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok || t.OnTabClick == nil {
		return EventIgnored
	}
	if mev.Buttons()&tcell.Button1 == 0 {
		return EventIgnored
	}
	r := t.GetRect()
	mx, my := mev.Position()
	if my < r.Y || my >= r.Y+r.H || mx < r.X || mx >= r.X+r.W {
		return EventIgnored
	}
	localX := mx - r.X + t.ScrollOffset
	for i, s := range t.tabSpans {
		if localX >= s.start && localX < s.end {
			t.OnTabClick(i)
			return EventConsumed
		}
	}
	return EventIgnored
}
