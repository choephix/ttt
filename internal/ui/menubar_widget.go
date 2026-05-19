package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type MenuItem struct {
	Name string
}

type MenuBarWidget struct {
	BaseWidget
	Items    []MenuItem
	Selected int
	OnSelect func(index int)
}

func NewMenuBarWidget(items []MenuItem) *MenuBarWidget {
	return &MenuBarWidget{
		Items:    items,
		Selected: -1,
	}
}

func (m *MenuBarWidget) Focusable() bool { return true }

func (m *MenuBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()

	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: term.StyleMenuBar})
	}

	x := 1
	for i, item := range m.Items {
		style := term.StyleMenuBar
		if i == m.Selected {
			style = term.StyleMenuBarActive
		}

		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
		x++
		for _, ch := range item.Name {
			if x < w {
				surface.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
				x++
			}
		}
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
		x++
		x++
	}
}

func (m *MenuBarWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyLeft:
		if m.Selected > 0 {
			m.Selected--
		}
		return EventConsumed
	case tcell.KeyRight:
		if m.Selected < len(m.Items)-1 {
			m.Selected++
		}
		return EventConsumed
	case tcell.KeyEnter:
		if m.OnSelect != nil && m.Selected >= 0 {
			m.OnSelect(m.Selected)
		}
		return EventConsumed
	}

	return EventIgnored
}
