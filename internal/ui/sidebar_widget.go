package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type SidebarWidget struct {
	BaseWidget
	TabbedPanel
	Visible    bool
	MoreButton *MoreButtonWidget
	OnSwitch   func(id string)
}

func NewSidebarWidget() *SidebarWidget {
	s := &SidebarWidget{
		TabbedPanel: NewTabbedPanel(),
		Visible:     true,
		MoreButton:  NewMoreButtonWidget(),
	}
	s.InitTabClick()
	return s
}

func (s *SidebarWidget) Focusable() bool { return true }

func (s *SidebarWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	r := s.GetRect()

	tabH := 2
	if h <= tabH {
		return
	}

	s.TabBar.Borders = s.Borders
	s.TabBar.SetRect(Rect{X: r.X, Y: r.Y, W: r.W, H: 1})
	tabSurface := surface.Sub(Rect{X: 0, Y: 0, W: w, H: 1})
	s.TabBar.Render(tabSurface)

	if s.MoreButton != nil && w >= 5 {
		s.MoreButton.SetRect(Rect{X: r.X + w - 4, Y: r.Y, W: 3, H: 1})
		moreSurface := surface.Sub(Rect{X: w - 4, Y: 0, W: 3, H: 1})
		s.MoreButton.Render(moreSurface)
	}

	bs := term.StyleBorder
	horizontal := '─'
	if s.Borders != nil {
		horizontal = s.Borders.Horizontal
	}
	for x := 0; x < w; x++ {
		surface.SetCell(x, 1, term.Cell{Ch: horizontal, Style: bs})
	}

	active := s.ActiveWidget()
	if active != nil {
		contentH := h - tabH
		active.SetRect(Rect{X: r.X, Y: r.Y + tabH, W: r.W, H: contentH})
		sub := surface.Sub(Rect{X: 0, Y: tabH, W: w, H: contentH})
		active.Render(sub)
	}
}

func (s *SidebarWidget) HandleEvent(ev tcell.Event) EventResult {
	if s.MoreButton != nil {
		if s.MoreButton.HandleEvent(ev) == EventConsumed {
			return EventConsumed
		}
	}
	if tev, ok := ev.(*tcell.EventMouse); ok {
		if tev.Buttons()&tcell.Button1 != 0 {
			_, my := tev.Position()
			r := s.GetRect()
			if my == r.Y {
				return s.TabBar.HandleEvent(ev)
			}
		}
	}
	active := s.ActiveWidget()
	if active != nil {
		return active.HandleEvent(ev)
	}
	return EventIgnored
}
