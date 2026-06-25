package ui

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type surfaceAdapter struct {
	surface *RenderSurface
}

func (a *surfaceAdapter) Size() (int, int)    { return a.surface.Size() }
func (a *surfaceAdapter) Fill(c term.Cell)     { a.surface.Fill(c) }
func (a *surfaceAdapter) SetCell(x, y int, c term.Cell) { a.surface.SetCell(x, y, c) }
func (a *surfaceAdapter) DrawText(x, y int, text string, maxW int, style term.Style) int {
	return a.surface.DrawText(x, y, text, maxW, style)
}
func (a *surfaceAdapter) ClearRect(x, y, w, h int, style term.Style) {
	a.surface.ClearRect(x, y, w, h, style)
}
func (a *surfaceAdapter) DrawBorder(x, y, w, h int, b term.BorderSet, style term.Style) {
	a.surface.DrawBorder(x, y, w, h, b, style)
}
func (a *surfaceAdapter) Sub(r widgets.Rect) widgets.Surface {
	return &surfaceAdapter{surface: a.surface.Sub(Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})}
}

type WidgetAdapter struct {
	BaseWidget
	W    widgets.Widget
	focus *widgets.FocusManager
}

func NewWidgetAdapter(w widgets.Widget) *WidgetAdapter {
	wa := &WidgetAdapter{W: w, focus: widgets.NewFocusManager()}
	wa.focus.Collect(w)
	wa.wireTabbedCallbacks(w)
	return wa
}

func (a *WidgetAdapter) wireTabbedCallbacks(w widgets.Widget) {
	switch v := w.(type) {
	case *widgets.TabbedWidget:
		v.OnChange = func(int) { a.RebuildFocus() }
		for _, child := range v.Children {
			a.wireTabbedCallbacks(child)
		}
	case *widgets.VStackWidget:
		for _, child := range v.Children {
			a.wireTabbedCallbacks(child)
		}
	case *widgets.HStackWidget:
		for _, child := range v.Children {
			a.wireTabbedCallbacks(child)
		}
	case *widgets.BoxWidget:
		if v.Child != nil {
			a.wireTabbedCallbacks(v.Child)
		}
	}
}

func (a *WidgetAdapter) Focusable() bool { return true }

func (a *WidgetAdapter) Render(surface *RenderSurface) {
	r := a.GetRect()
	a.W.SetRect(widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.W.Render(&surfaceAdapter{surface: surface})
}

func (a *WidgetAdapter) RebuildFocus() {
	a.focus.Collect(a.W)
}

func (a *WidgetAdapter) RewireTabbedCallbacks() {
	a.wireTabbedCallbacks(a.W)
}

func (a *WidgetAdapter) CursorPosition() (int, int, bool) {
	if fw := a.focus.Focused(); fw != nil {
		if cp, ok := fw.(widgets.CursorPositioner); ok {
			return cp.CursorPosition()
		}
	}
	return 0, 0, false
}

func (a *WidgetAdapter) HandleEvent(ev tcell.Event) EventResult {
	if a.focus.HandleEvent(ev) {
		return EventConsumed
	}
	if a.W.HandleEvent(ev) {
		return EventConsumed
	}
	return EventIgnored
}
