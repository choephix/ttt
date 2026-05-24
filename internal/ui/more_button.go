package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type MoreButtonWidget struct {
	BaseWidget
	OnClick func(screenX, screenY int)
}

func NewMoreButtonWidget() *MoreButtonWidget {
	return &MoreButtonWidget{}
}

func (m *MoreButtonWidget) Focusable() bool { return false }

func (m *MoreButtonWidget) Render(surface *RenderSurface) {
	surface.SetCell(0, 0, term.Cell{Ch: ' ', Style: term.StyleInactiveTab})
	surface.SetCell(1, 0, term.Cell{Ch: '⋮', Style: term.StyleInactiveTab})
	surface.SetCell(2, 0, term.Cell{Ch: ' ', Style: term.StyleInactiveTab})
}

func (m *MoreButtonWidget) HandleEvent(ev tcell.Event) EventResult {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return EventIgnored
	}
	if mev.Buttons()&tcell.Button1 == 0 {
		return EventIgnored
	}
	r := m.GetRect()
	mx, my := mev.Position()
	if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
		if m.OnClick != nil {
			m.OnClick(r.X+1, r.Y+r.H)
		}
		return EventConsumed
	}
	return EventIgnored
}
