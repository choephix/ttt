package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type sidebarEntry struct {
	ID    string
	Title string
	W     Widget
}

type SidebarWidget struct {
	BaseWidget
	panels      []sidebarEntry
	ActivePanel string
	Visible     bool
	Borders     *term.BorderSet
	TabBar      *PanelTabBarWidget
	MoreButton  *MoreButtonWidget
	OnSwitch    func(id string)
}

func NewSidebarWidget() *SidebarWidget {
	tabBar := NewPanelTabBarWidget()
	return &SidebarWidget{
		Visible:    true,
		TabBar:     tabBar,
		MoreButton: NewMoreButtonWidget(),
	}
}

func (s *SidebarWidget) AddPanel(id, title string, w Widget) {
	s.panels = append(s.panels, sidebarEntry{ID: id, Title: title, W: w})
	if s.ActivePanel == "" {
		s.ActivePanel = id
	}
	s.syncTabs()
}

func (s *SidebarWidget) SetActivePanel(id string) {
	for _, p := range s.panels {
		if p.ID == id {
			s.ActivePanel = id
			s.syncTabs()
			return
		}
	}
}

func (s *SidebarWidget) ActiveWidget() Widget {
	for _, p := range s.panels {
		if p.ID == s.ActivePanel {
			return p.W
		}
	}
	return nil
}

func (s *SidebarWidget) Focusable() bool { return true }

func (s *SidebarWidget) syncTabs() {
	var tabs []Tab
	for _, p := range s.panels {
		tabs = append(tabs, Tab{
			Name:   p.Title,
			Active: p.ID == s.ActivePanel,
		})
	}
	s.TabBar.SetTabs(tabs)
}

func (s *SidebarWidget) handleTabClick(mx int) {
	x := 0
	for _, p := range s.panels {
		label := " " + p.Title + " "
		end := x + len([]rune(label))
		if mx >= x && mx < end {
			s.ActivePanel = p.ID
			s.syncTabs()
			if s.OnSwitch != nil {
				s.OnSwitch(p.ID)
			}
			return
		}
		x = end
	}
}

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
			mx, my := tev.Position()
			r := s.GetRect()
			if my == r.Y {
				s.handleTabClick(mx - r.X)
				return EventConsumed
			}
		}
	}
	active := s.ActiveWidget()
	if active != nil {
		return active.HandleEvent(ev)
	}
	return EventIgnored
}
