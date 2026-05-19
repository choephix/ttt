package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ResizeHandleWidget struct {
	BaseWidget
	Dragging  bool
	OnResize  func(deltaX int)
	lastMouseX int
}

func NewResizeHandleWidget() *ResizeHandleWidget {
	return &ResizeHandleWidget{}
}

func (r *ResizeHandleWidget) Focusable() bool { return false }

func (r *ResizeHandleWidget) Render(surface *RenderSurface) {
	_, h := surface.Size()
	for y := 0; y < h; y++ {
		surface.SetCell(0, y, term.Cell{Ch: '│', Style: term.StyleResizeHandle})
	}
}

func (r *ResizeHandleWidget) HandleEvent(ev tcell.Event) EventResult {
	return EventIgnored
}
