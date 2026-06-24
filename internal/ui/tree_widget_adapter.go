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

type CardWidgetAdapter struct {
	BaseWidget
	Box *widgets.CardWidget
}

func NewCardWidgetAdapter(box *widgets.CardWidget) *CardWidgetAdapter {
	return &CardWidgetAdapter{Box: box}
}

func (a *CardWidgetAdapter) Focusable() bool { return true }

func (a *CardWidgetAdapter) Render(surface *RenderSurface) {
	r := a.GetRect()
	a.Box.SetRect(widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.Box.Render(&surfaceAdapter{surface: surface})
}

func (a *CardWidgetAdapter) HandleEvent(ev tcell.Event) EventResult {
	if a.Box.HandleEvent(ev) {
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

type TitleWidgetAdapter struct {
	BaseWidget
	Section *widgets.TitleWidget
}

func NewTitleWidgetAdapter(section *widgets.TitleWidget) *TitleWidgetAdapter {
	return &TitleWidgetAdapter{Section: section}
}

func (a *TitleWidgetAdapter) Focusable() bool { return false }

func (a *TitleWidgetAdapter) Render(surface *RenderSurface) {
	r := a.GetRect()
	a.Section.SetRect(widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.Section.Render(&surfaceAdapter{surface: surface})
}

func (a *TitleWidgetAdapter) HandleEvent(ev tcell.Event) EventResult {
	if a.Section.HandleEvent(ev) {
		return EventConsumed
	}
	return EventIgnored
}
