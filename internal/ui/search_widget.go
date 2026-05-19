package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type SearchWidget struct {
	BaseWidget
	Query   string
	Results []string
}

func NewSearchWidget() *SearchWidget {
	return &SearchWidget{}
}

func (s *SearchWidget) Focusable() bool { return true }

func (s *SearchWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	// Header
	header := "SEARCH"
	for i, ch := range header {
		if i < w {
			surface.SetCell(i, 0, term.Cell{Ch: ch, Style: term.StyleSidebarHeader})
		}
	}
	for i := len(header); i < w; i++ {
		surface.SetCell(i, 0, term.Cell{Ch: ' ', Style: term.StyleSidebarHeader})
	}

	if h < 3 {
		return
	}

	// Search input
	prompt := "> " + s.Query + "_"
	for i, ch := range prompt {
		if i < w {
			surface.SetCell(i, 2, term.Cell{Ch: ch})
		}
	}

	// Results
	for i, result := range s.Results {
		y := 4 + i
		if y >= h {
			break
		}
		for x, ch := range result {
			if x >= w {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: term.StyleSidebarItem})
		}
	}
}

func (s *SearchWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyRune:
		s.Query += string(kev.Rune())
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(s.Query) > 0 {
			runes := []rune(s.Query)
			s.Query = string(runes[:len(runes)-1])
		}
		return EventConsumed
	}

	return EventIgnored
}
