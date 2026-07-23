package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v3"
)

type MenuItem struct {
	Name string
}

type MenuBarWidget struct {
	BaseWidget
	Items      []MenuItem
	Selected   int
	OnSelect   func(index int)
	itemSpans  []MenuItemSpan
	wasPressed bool
}

func NewMenuBarWidget(items []MenuItem) *MenuBarWidget {
	return &MenuBarWidget{
		Items:    items,
		Selected: -1,
	}
}

type MenuItemSpan struct {
	Start, End int
}

func (m *MenuBarWidget) ItemSpans() []MenuItemSpan { return m.itemSpans }

func (m *MenuBarWidget) Height() int     { return 1 }
func (m *MenuBarWidget) Focusable() bool { return true }

func (m *MenuBarWidget) Render(surface Surface) {
	w, _ := surface.Size()

	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: term.StyleMenuBar})
	}

	m.itemSpans = make([]MenuItemSpan, len(m.Items))
	x := 1
	for i, item := range m.Items {
		style := term.StyleMenuBar
		if i == m.Selected {
			style = term.StyleMenuBarActive
		}

		startX := x
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
		m.itemSpans[i] = MenuItemSpan{startX, x}
		x++
	}
}

func (m *MenuBarWidget) ItemAnchorX(index int) int {
	if index >= 0 && index < len(m.itemSpans) {
		return m.itemSpans[index].Start
	}
	return 0
}

func (m *MenuBarWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		btn := tev.Buttons()
		if btn == tcell.ButtonNone {
			m.wasPressed = false
			return EventIgnored
		}
		if btn&tcell.Button1 != 0 && !m.wasPressed {
			m.wasPressed = true
			r := m.GetRect()
			mx, my := tev.Position()
			if my == r.Y {
				localX := mx - r.X
				for i, span := range m.itemSpans {
					if localX >= span.Start && localX < span.End {
						m.Selected = i
						if m.OnSelect != nil {
							m.OnSelect(i)
						}
						return EventConsumed
					}
				}
			}
		}
	case *tcell.EventKey:
		switch tev.Key() {
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
	}

	return EventIgnored
}
