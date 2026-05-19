package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type SidebarWidget struct {
	BaseWidget
	Panels      map[string]Widget
	ActivePanel string
	Visible     bool
	Title       string
	Borders     *term.BorderSet
}

func NewSidebarWidget() *SidebarWidget {
	return &SidebarWidget{
		Panels:  make(map[string]Widget),
		Visible: true,
	}
}

func (s *SidebarWidget) AddPanel(id string, w Widget) {
	s.Panels[id] = w
	if s.ActivePanel == "" {
		s.ActivePanel = id
	}
}

func (s *SidebarWidget) SetActivePanel(id string) {
	if _, ok := s.Panels[id]; ok {
		s.ActivePanel = id
	}
}

func (s *SidebarWidget) ActiveWidget() Widget {
	if w, ok := s.Panels[s.ActivePanel]; ok {
		return w
	}
	return nil
}

func (s *SidebarWidget) Focusable() bool { return true }

func (s *SidebarWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	r := s.GetRect()
	titleH := 0

	if s.Title != "" && h > 2 {
		titleH = 2
		bs := term.StyleBorder
		horizontal := '─'
		if s.Borders != nil {
			horizontal = s.Borders.Horizontal
		}
		for x := 0; x < w; x++ {
			surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: term.StyleSidebarHeader})
			surface.SetCell(x, 1, term.Cell{Ch: horizontal, Style: bs})
		}
		for i, ch := range s.Title {
			if i+1 >= w {
				break
			}
			surface.SetCell(i+1, 0, term.Cell{Ch: ch, Style: term.StyleSidebarHeader})
		}
		if w >= 2 {
			surface.SetCell(w-1, 0, term.Cell{Ch: '…', Style: term.StyleSidebarHeader})
		}
	}

	active := s.ActiveWidget()
	if active != nil && h > titleH {
		active.SetRect(Rect{X: r.X, Y: r.Y + titleH, W: r.W, H: r.H - titleH})
		sub := surface.Sub(Rect{X: 0, Y: titleH, W: w, H: h - titleH})
		active.Render(sub)
	}
}

func (s *SidebarWidget) HandleEvent(ev tcell.Event) EventResult {
	active := s.ActiveWidget()
	if active != nil {
		return active.HandleEvent(ev)
	}
	return EventIgnored
}
