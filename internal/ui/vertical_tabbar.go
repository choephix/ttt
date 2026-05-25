package ui

import (
	"fmt"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

// VerticalTabBarWidth is border (1) + content (4)
const VerticalTabBarWidth = 5

type VerticalTabBar struct {
	BaseWidget
	Count    int
	Active   int
	OnSelect func(index int)
	Borders  *term.BorderSet
}

func NewVerticalTabBar() *VerticalTabBar {
	return &VerticalTabBar{}
}

func (v *VerticalTabBar) Focusable() bool { return false }

func (v *VerticalTabBar) Render(surface *RenderSurface) {
	_, h := surface.Size()

	vertical := '│'
	if v.Borders != nil {
		vertical = v.Borders.Vertical
	}

	// Clear and draw left border
	for y := 0; y < h; y++ {
		surface.SetCell(0, y, term.Cell{Ch: vertical, Style: term.StyleBorder})
		for x := 1; x < VerticalTabBarWidth; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: term.StyleDefault})
		}
	}

	for i := 0; i < v.Count; i++ {
		if i >= h {
			break
		}
		style := term.StyleInactiveTab
		if i == v.Active {
			style = term.StyleActiveTab
		}
		n := i + 1
		var label string
		if n < 10 {
			label = fmt.Sprintf("[>%d]", n)
		} else {
			label = fmt.Sprintf("[%d]", n)
		}
		for x, ch := range label {
			if x+1 < VerticalTabBarWidth {
				surface.SetCell(x+1, i, term.Cell{Ch: ch, Style: style})
			}
		}
	}
}

func (v *VerticalTabBar) HandleEvent(ev tcell.Event) EventResult {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return EventIgnored
	}
	if mev.Buttons()&tcell.Button1 == 0 {
		return EventIgnored
	}
	_, my := mev.Position()
	r := v.GetRect()
	ly := my - r.Y

	if ly >= 0 && ly < v.Count && v.OnSelect != nil {
		v.OnSelect(ly)
		return EventConsumed
	}
	return EventConsumed
}
