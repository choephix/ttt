package widgets

import (
	"github.com/gdamore/tcell/v2"
)

type FocusManager struct {
	items   []FocusableWidget
	focused int
}

func NewFocusManager() *FocusManager {
	return &FocusManager{focused: -1}
}

func (fm *FocusManager) Collect(w Widget) {
	for _, item := range fm.items {
		item.SetFocused(false)
	}
	fm.items = nil
	fm.focused = -1
	collectFocusable(w, &fm.items)
	if len(fm.items) > 0 {
		fm.focused = 0
		fm.items[0].SetFocused(true)
	}
}

func collectFocusable(w Widget, out *[]FocusableWidget) {
	if fw, ok := w.(FocusableWidget); ok && fw.Focusable() {
		*out = append(*out, fw)
	}
	switch v := w.(type) {
	case *VStackWidget:
		for _, child := range v.Children {
			collectFocusable(child, out)
		}
	case *HStackWidget:
		for _, child := range v.Children {
			collectFocusable(child, out)
		}
	case *BoxWidget:
		if v.Child != nil {
			collectFocusable(v.Child, out)
		}
	case *TabbedWidget:
		collectFocusable(v.Tabs, out)
		if c := v.ActiveChild(); c != nil {
			collectFocusable(c, out)
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

func (fm *FocusManager) FocusByClick(mx, my int) {
	for i, fw := range fm.items {
		r := fw.GetRect()
		if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
			fm.setFocus(i)
			return
		}
	}
}

func (fm *FocusManager) setFocus(idx int) {
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		fm.items[fm.focused].SetFocused(false)
	}
	fm.focused = idx
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		fm.items[fm.focused].SetFocused(true)
	}
}

func (fm *FocusManager) ItemCount() int              { return len(fm.items) }
func (fm *FocusManager) Items() []FocusableWidget     { return fm.items }

func (fm *FocusManager) Focused() Widget {
	if fm.focused >= 0 && fm.focused < len(fm.items) {
		return fm.items[fm.focused]
	}
	return nil
}

func (fm *FocusManager) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		if tev.Key() == tcell.KeyTab {
			fm.FocusNext()
			return true
		}
		if tev.Key() == tcell.KeyBacktab {
			fm.FocusPrev()
			return true
		}
		if fw := fm.Focused(); fw != nil {
			return fw.HandleEvent(ev)
		}
	case *tcell.EventMouse:
		if tev.Buttons()&tcell.Button1 != 0 {
			fm.FocusByClick(tev.Position())
		}
	}
	return false
}
