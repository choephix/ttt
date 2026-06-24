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

type TreeWidgetAdapter struct {
	BaseWidget
	Tree *widgets.TreeWidget
}

func NewTreeWidgetAdapter(tree *widgets.TreeWidget) *TreeWidgetAdapter {
	return &TreeWidgetAdapter{Tree: tree}
}

func (a *TreeWidgetAdapter) Focusable() bool { return true }

func (a *TreeWidgetAdapter) Render(surface *RenderSurface) {
	r := a.GetRect()
	a.Tree.SetRect(widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.Tree.Render(&surfaceAdapter{surface: surface})
}

func (a *TreeWidgetAdapter) HandleEvent(ev tcell.Event) EventResult {
	if a.Tree.HandleEvent(ev) {
		return EventConsumed
	}
	return EventIgnored
}


type BoxWidgetAdapter struct {
	BaseWidget
	Box *widgets.BoxWidget
}

func NewBoxWidgetAdapter(box *widgets.BoxWidget) *BoxWidgetAdapter {
	return &BoxWidgetAdapter{Box: box}
}

func (a *BoxWidgetAdapter) Focusable() bool { return true }

func (a *BoxWidgetAdapter) Render(surface *RenderSurface) {
	r := a.GetRect()
	a.Box.SetRect(widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.Box.Render(&surfaceAdapter{surface: surface})
}

func (a *BoxWidgetAdapter) HandleEvent(ev tcell.Event) EventResult {
	if a.Box.HandleEvent(ev) {
		return EventConsumed
	}
	return EventIgnored
}

type WidgetAdapter struct {
	BaseWidget
	W widgets.Widget
}

func NewWidgetAdapter(w widgets.Widget) *WidgetAdapter {
	return &WidgetAdapter{W: w}
}

func (a *WidgetAdapter) Focusable() bool { return true }

func (a *WidgetAdapter) Render(surface *RenderSurface) {
	r := a.GetRect()
	a.W.SetRect(widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.W.Render(&surfaceAdapter{surface: surface})
}

func (a *WidgetAdapter) HandleEvent(ev tcell.Event) EventResult {
	if a.W.HandleEvent(ev) {
		return EventConsumed
	}
	return EventIgnored
}
