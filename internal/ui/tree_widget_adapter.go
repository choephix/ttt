package ui

import (
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type WidgetAdapter struct {
	BaseWidget
	W      widgets.Widget
	focus  *widgets.FocusManager
	popups []widgets.PopupRenderer
}

func NewWidgetAdapter(w widgets.Widget) *WidgetAdapter {
	wa := &WidgetAdapter{W: w, focus: widgets.NewFocusManager()}
	wa.focus.Collect(w)
	wa.collectPopups()
	wa.wireTabbedCallbacks(w)
	return wa
}

// EnableScrollIntoView keeps the focused widget on screen while tabbing through
// content taller than its scroll view. Opt-in so existing panels and dialogs
// keep their current scrolling behaviour.
func (a *WidgetAdapter) EnableScrollIntoView() {
	a.focus.OnFocusChange = func(fw widgets.FocusableWidget) {
		widgets.ScrollIntoView(a.W, fw)
	}
}

// Popup-bearing widgets are cached rather than rediscovered each frame, and
// refreshed alongside focus whenever the tree changes.
func (a *WidgetAdapter) collectPopups() {
	a.popups = nil
	var walk func(widgets.Widget)
	walk = func(w widgets.Widget) {
		if pr, ok := w.(widgets.PopupRenderer); ok {
			a.popups = append(a.popups, pr)
		}
		if cw, ok := w.(widgets.ContainerWidget); ok {
			for _, child := range cw.WidgetChildren() {
				walk(child)
			}
		}
	}
	walk(a.W)
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

	// Popups draw after the tree so they overlay it. This covers any popup-bearing
	// widget, not just the focused one, because a click can leave focus on an
	// ancestor (a scroll view) while opening a popup on a leaf.
	bounds := Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
	for _, pr := range a.popups {
		if pb, ok := pr.(widgets.PopupBounder); ok {
			pb.SetPopupBounds(bounds)
		}
		if !pr.HasPopup() {
			continue
		}
		rect := pr.PopupRect()
		pr.RenderPopup(surface.Sub(Rect{
			X: rect.X - bounds.X, Y: rect.Y - bounds.Y, W: rect.W, H: rect.H,
		}))
	}
}

func (a *WidgetAdapter) RebuildFocus() {
	a.focus.Collect(a.W)
	a.collectPopups()
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
