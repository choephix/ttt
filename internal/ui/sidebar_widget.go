package ui

import "github.com/gdamore/tcell/v2"

type SidebarWidget struct {
	BaseWidget
	Panels      map[string]Widget
	ActivePanel string
	Visible     bool
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
	active := s.ActiveWidget()
	if active != nil {
		r := s.GetRect()
		active.SetRect(r)
		active.Render(surface)
	}
}

func (s *SidebarWidget) HandleEvent(ev tcell.Event) EventResult {
	active := s.ActiveWidget()
	if active != nil {
		return active.HandleEvent(ev)
	}
	return EventIgnored
}
