package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ResizeHandleWidget struct {
	BaseWidget
	Borders *term.BorderSet
}

func NewResizeHandleWidget(borders *term.BorderSet) *ResizeHandleWidget {
	return &ResizeHandleWidget{Borders: borders}
}

func (r *ResizeHandleWidget) Focusable() bool { return false }

func (r *ResizeHandleWidget) Render(surface *RenderSurface) {
	_, h := surface.Size()
	ch := '║'
	if r.Borders != nil {
		ch = r.Borders.Vertical
	}
	for y := 0; y < h; y++ {
		surface.SetCell(0, y, term.Cell{Ch: ch, Style: term.StyleResizeHandle})
	}
}

func (r *ResizeHandleWidget) HandleEvent(ev tcell.Event) EventResult {
	return EventIgnored
}
