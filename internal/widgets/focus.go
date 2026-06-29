package widgets

import (
	"github.com/gdamore/tcell/v2"
)

type FocusManager struct {
	items   []FocusableWidget
	focused int
	active  bool
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
	if fm.active && fm.focused >= 0 && fm.focused < len(fm.items) {
		fm.items[fm.focused].SetFocused(true)
	}
}

func (fm *FocusManager) ItemCount() int              { return len(fm.items) }
func (fm *FocusManager) Items() []FocusableWidget     { return fm.items }
func (fm *FocusManager) HasNext() bool                { return fm.focused < len(fm.items)-1 }
func (fm *FocusManager) HasPrev() bool                { return fm.focused > 0 }

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
		for _, fw := range fm.items {
			r := fw.GetRect()
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
