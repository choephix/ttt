package ui

import (
	"macro/internal/term"
	"macro/internal/view"
)

type StatusBarWidget struct {
	BaseWidget
	Status *view.StatusBar
}

func NewStatusBarWidget(status *view.StatusBar) *StatusBarWidget {
	return &StatusBarWidget{Status: status}
}

func (s *StatusBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	bar := s.Status.RenderStatusBar(w)
	runes := []rune(bar)
	for i := 0; i < w; i++ {
		ch := ' '
		if i < len(runes) {
			ch = runes[i]
		}
		surface.SetCell(i, 0, term.Cell{Ch: ch, Style: term.StyleStatusBar})
	}
}
