package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type BottomPanelWidget struct {
	BaseWidget
	TabbedPanel
	Visible bool
}

func NewBottomPanelWidget(borders *term.BorderSet) *BottomPanelWidget {
	tp := NewTabbedPanel()
	tp.Borders = borders
	tp.TabBar.Borders = borders
	bp := &BottomPanelWidget{
		TabbedPanel: tp,
		Visible:     true,
	}
	bp.InitTabClick()
	return bp
}

func (bp *BottomPanelWidget) Focusable() bool { return true }

func (bp *BottomPanelWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	r := bp.GetRect()

	if h < 3 {
		return
	}

	bs := term.StyleBorder
	horizontal := '─'
	if bp.Borders != nil {
		horizontal = bp.Borders.Horizontal
	}

	bp.TabBar.SetRect(Rect{X: r.X, Y: r.Y, W: r.W, H: 1})
	tabSurface := surface.Sub(Rect{X: 0, Y: 0, W: w, H: 1})
	bp.TabBar.Render(tabSurface)

	for x := 0; x < w; x++ {
		surface.SetCell(x, 1, term.Cell{Ch: horizontal, Style: bs})
	}

	contentH := h - 2
	active := bp.ActiveWidget()
	if active != nil && contentH > 0 {
		active.SetRect(Rect{X: r.X, Y: r.Y + 2, W: r.W, H: contentH})
		contentSurface := surface.Sub(Rect{X: 0, Y: 2, W: w, H: contentH})
		active.Render(contentSurface)
	}
}

func (bp *BottomPanelWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		_, my := mev.Position()
		r := bp.GetRect()
		if my == r.Y {
			return bp.TabBar.HandleEvent(ev)
		}
	}
	active := bp.ActiveWidget()
	if active != nil {
		return active.HandleEvent(ev)
	}
	return EventIgnored
}
