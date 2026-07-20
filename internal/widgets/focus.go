package widgets

import (
	"github.com/gdamore/tcell/v2"
)

type FocusManager struct {
	items   []FocusableWidget
	focused int
	active  bool
	root    Widget
	// OnFocusChange is called after focus moves (e.g. to scroll the widget into view).
	OnFocusChange func(w FocusableWidget)
}

func NewFocusManager() *FocusManager {
	return &FocusManager{focused: -1}
}

func (fm *FocusManager) SetActive(active bool) {
	fm.active = active
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		fm.items[fm.focused].SetFocused(active)
	}
}

func (fm *FocusManager) Collect(w Widget) {
	var prev FocusableWidget
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		prev = fm.items[fm.focused]
	}
	for _, item := range fm.items {
		item.SetFocused(false)
	}
	fm.items = nil
	fm.focused = -1
	fm.root = w
	collectFocusable(w, &fm.items)
	if len(fm.items) > 0 {
		fm.focused = 0
		if prev != nil {
			for i, item := range fm.items {
				if item == prev {
					fm.focused = i
					break
				}
			}
		}
		if fm.active {
			fm.items[fm.focused].SetFocused(true)
		}
	}
}

func collectFocusable(w Widget, out *[]FocusableWidget) {
	if fw, ok := w.(FocusableWidget); ok && fw.Focusable() {
		*out = append(*out, fw)
	}
	if cw, ok := w.(ContainerWidget); ok {
		for _, child := range cw.WidgetChildren() {
			collectFocusable(child, out)
		}
	}
}

func (fm *FocusManager) FocusNext() {
	if len(fm.items) == 0 {
		return
	}
	fm.setFocus((fm.focused + 1) % len(fm.items))
}

func (fm *FocusManager) FocusPrev() {
	if len(fm.items) == 0 {
		return
	}
	next := fm.focused - 1
	if next < 0 {
		next = len(fm.items) - 1
	}
	fm.setFocus(next)
}

// Items are collected pre-order, so a click inside a focusable container (a
// scroll view, say) matches the container before the control the user aimed at.
// The last match is the innermost one.
func (fm *FocusManager) FocusByClick(mx, my int) {
	// A popup floats over unrelated rows, so rect matching would hand focus to
	// whatever sits underneath and close the popup before the click lands.
	if pr, ok := fm.Focused().(PopupRenderer); ok && pr.HasPopup() {
		r := pr.PopupRect()
		if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
			return
		}
	}

	hit := -1
	for i, fw := range fm.items {
		r := VisibleRect(fm.root, fw)
		if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
			hit = i
		}
	}
	if hit >= 0 {
		fm.setFocus(hit)
	}
}

func (fm *FocusManager) setFocus(idx int) {
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		fm.items[fm.focused].SetFocused(false)
	}
	fm.focused = idx
	if fm.active && fm.focused >= 0 && fm.focused < len(fm.items) {
		fm.items[fm.focused].SetFocused(true)
		if fm.OnFocusChange != nil {
			fm.OnFocusChange(fm.items[fm.focused])
		}
	}
}

// ScrollIntoView scrolls every scroll view between root and target to make the target visible.
func ScrollIntoView(root, target Widget) {
	var svs []*ScrollViewWidget
	if !collectScrollPath(root, target, &svs) {
		return
	}
	r := target.GetRect()
	for i := len(svs) - 1; i >= 0; i-- {
		sv := svs[i]
		ox, oy := sv.viewportOrigin()
		cx := r.X - ox + sv.scrollX
		cy := r.Y - oy + sv.scrollY
		sv.EnsureVisible(cx, cy+r.H-1)
		sv.EnsureVisible(cx, cy)
	}
}

// VisibleRect returns the target's screen rect clipped by enclosing scroll view viewports.
func VisibleRect(root, target Widget) Rect {
	r := target.GetRect()
	if root == nil {
		return r
	}
	var svs []*ScrollViewWidget
	if !collectScrollPath(root, target, &svs) {
		return r
	}
	for _, sv := range svs {
		ox, oy := sv.viewportOrigin()
		contentW, contentH := 0, 0
		if sv.Child != nil {
			contentW, contentH = sv.Child.ScrollSize()
		}
		innerW := sv.rect.W - sv.BoxOverheadW()
		innerH := sv.rect.H - sv.BoxOverheadH()
		viewW, viewH := sv.viewportSize(innerW, innerH, contentW, contentH)
		r = intersectRects(r, Rect{X: ox, Y: oy, W: viewW, H: viewH})
		if r.W <= 0 || r.H <= 0 {
			return Rect{}
		}
	}
	return r
}

func intersectRects(a, b Rect) Rect {
	x1 := max(a.X, b.X)
	y1 := max(a.Y, b.Y)
	x2 := min(a.X+a.W, b.X+b.W)
	y2 := min(a.Y+a.H, b.Y+b.H)
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

func collectScrollPath(w, target Widget, svs *[]*ScrollViewWidget) bool {
	if w == target {
		return true
	}
	if sv, ok := w.(*ScrollViewWidget); ok {
		*svs = append(*svs, sv)
		if cw, ok := w.(ContainerWidget); ok {
			for _, child := range cw.WidgetChildren() {
				if collectScrollPath(child, target, svs) {
					return true
				}
			}
		}
		*svs = (*svs)[:len(*svs)-1]
		return false
	}
	if cw, ok := w.(ContainerWidget); ok {
		for _, child := range cw.WidgetChildren() {
			if collectScrollPath(child, target, svs) {
				return true
			}
		}
	}
	return false
}

func (fm *FocusManager) ItemCount() int           { return len(fm.items) }
func (fm *FocusManager) Items() []FocusableWidget { return fm.items }
func (fm *FocusManager) HasNext() bool            { return fm.focused < len(fm.items)-1 }
func (fm *FocusManager) HasPrev() bool            { return fm.focused > 0 }

func (fm *FocusManager) Focused() Widget {
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		return fm.items[fm.focused]
	}
	return nil
}

func (fm *FocusManager) HandleEvent(ev tcell.Event) EventResult {
	if len(fm.items) == 0 {
		return EventIgnored
	}
	switch tev := ev.(type) {
	case *tcell.EventKey:
		if tev.Key() == tcell.KeyTab && len(fm.items) > 1 {
			fm.FocusNext()
			return EventConsumed
		}
		if tev.Key() == tcell.KeyBacktab && len(fm.items) > 1 {
			fm.FocusPrev()
			return EventConsumed
		}
		if fw := fm.Focused(); fw != nil {
			return fw.HandleEvent(ev)
		}
	case *tcell.EventMouse:
		if tev.Buttons()&tcell.Button1 != 0 {
			fm.FocusByClick(tev.Position())
		}
		mx, my := tev.Position()
		// An open popup covers rows that are not its owner's, so it claims clicks
		// inside its bounds before the rect scan below.
		if pr, ok := fm.Focused().(PopupRenderer); ok && pr.HasPopup() {
			r := pr.PopupRect()
			if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
				return pr.(FocusableWidget).HandleEvent(ev)
			}
		}
		for _, fw := range fm.items {
			r := VisibleRect(fm.root, fw)
			if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
				return fw.HandleEvent(ev)
			}
		}
		if fw := fm.Focused(); fw != nil {
			return fw.HandleEvent(ev)
		}
	}
	return EventIgnored
}
