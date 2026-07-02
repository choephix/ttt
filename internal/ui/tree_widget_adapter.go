package ui

import (
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type WidgetAdapter struct {
	BaseWidget
	W     widgets.Widget
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
	case *widgets.ScrollViewWidget:
		if v.Child != nil {
			a.wireTabbedCallbacks(v.Child)
		}
	}
}

func (a *WidgetAdapter) Inner() widgets.Widget { return a.W }

func (a *WidgetAdapter) Focusable() bool { return true }

func (a *WidgetAdapter) SetFocused(focused bool) {
	a.focus.SetActive(focused)
}

func (a *WidgetAdapter) Render(surface Surface) {
	r := a.GetRect()
	a.W.SetRect(Rect{X: r.X, Y: r.Y, W: r.W, H: r.H})
	a.W.Render(surface)

	if fw := a.focus.Focused(); fw != nil {
		if pr, ok := fw.(widgets.PopupRenderer); ok && pr.HasPopup() {
			rect := pr.PopupRect()
			pr.RenderPopup(surface.Sub(Rect{X: rect.X - r.X, Y: rect.Y - r.Y, W: rect.W, H: rect.H}))
		}
	}
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
	if result := a.focus.HandleEvent(ev); result != EventIgnored {
		return result
	}
	return a.W.HandleEvent(ev)
}
