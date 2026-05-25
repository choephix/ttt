package ui

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TerminalPanelWidget struct {
	BaseWidget
	TabBar  *VerticalTabBar
	widgets []Widget
	active  int
	Borders *term.BorderSet
}

func NewTerminalPanelWidget() *TerminalPanelWidget {
	tp := &TerminalPanelWidget{
		TabBar: NewVerticalTabBar(),
	}
	tp.TabBar.OnSelect = func(index int) {
		tp.SetActive(index)
	}
	return tp
}

func (tp *TerminalPanelWidget) Focusable() bool { return true }

func (tp *TerminalPanelWidget) WantsRawKeys() bool {
	if w := tp.ActiveWidget(); w != nil {
		if rk, ok := w.(RawKeyConsumer); ok {
			return rk.WantsRawKeys()
		}
	}
	return false
}

func (tp *TerminalPanelWidget) SetFocused(focused bool) {
	if w := tp.ActiveWidget(); w != nil {
		if setter, ok := w.(interface{ SetFocused(bool) }); ok {
			setter.SetFocused(focused)
		}
	}
}

func (tp *TerminalPanelWidget) AddTerminal(w Widget) {
	tp.widgets = append(tp.widgets, w)
	tp.active = len(tp.widgets) - 1
	tp.syncTabBar()
}

func (tp *TerminalPanelWidget) RemoveTerminal(index int) {
	if index < 0 || index >= len(tp.widgets) {
		return
	}
	tp.widgets = append(tp.widgets[:index], tp.widgets[index+1:]...)
	if tp.active >= len(tp.widgets) {
		tp.active = len(tp.widgets) - 1
	}
	if tp.active < 0 {
		tp.active = 0
	}
	tp.syncTabBar()
}

func (tp *TerminalPanelWidget) SetActive(index int) {
	if index >= 0 && index < len(tp.widgets) && tp.active != index {
		if old := tp.ActiveWidget(); old != nil {
			if setter, ok := old.(interface{ SetFocused(bool) }); ok {
				setter.SetFocused(false)
			}
		}
		tp.active = index
		if w := tp.ActiveWidget(); w != nil {
			if setter, ok := w.(interface{ SetFocused(bool) }); ok {
				setter.SetFocused(true)
			}
		}
		tp.syncTabBar()
	}
}

func (tp *TerminalPanelWidget) ActiveIndex() int {
	return tp.active
}

func (tp *TerminalPanelWidget) ActiveWidget() Widget {
	if tp.active >= 0 && tp.active < len(tp.widgets) {
		return tp.widgets[tp.active]
	}
	return nil
}

func (tp *TerminalPanelWidget) Count() int {
	return len(tp.widgets)
}

func (tp *TerminalPanelWidget) syncTabBar() {
	tp.TabBar.Count = len(tp.widgets)
	tp.TabBar.Active = tp.active
}

func (tp *TerminalPanelWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	r := tp.GetRect()

	if len(tp.widgets) == 0 {
		msg := "No terminals. Press + to create one."
		x := 1
		for _, ch := range msg {
			if x >= w {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: term.StyleMuted})
			x++
		}
		return
	}

	stripW := VerticalTabBarWidth
	contentW := w - stripW
	if contentW <= 0 {
		return
	}

	tp.TabBar.Borders = tp.Borders
	tp.TabBar.SetRect(Rect{X: r.X, Y: r.Y, W: stripW, H: h})
	tp.TabBar.Render(surface.Sub(Rect{X: 0, Y: 0, W: stripW, H: h}))

	active := tp.widgets[tp.active]
	active.SetRect(Rect{X: r.X + stripW, Y: r.Y, W: contentW, H: h})
	active.Render(surface.Sub(Rect{X: stripW, Y: 0, W: contentW, H: h}))
}

func (tp *TerminalPanelWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, _ := mev.Position()
			r := tp.GetRect()
			if mx-r.X < VerticalTabBarWidth {
				return tp.TabBar.HandleEvent(ev)
			}
		}
	}

	if w := tp.ActiveWidget(); w != nil {
		return w.HandleEvent(ev)
	}
	return EventIgnored
}
