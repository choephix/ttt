package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type Overlay struct {
	Widget Widget
	Modal  bool
}

type GlobalKeyBinding struct {
	Key      tcell.Key
	Mod      tcell.ModMask
	Rune     rune
	Handler  func()
}

type Root struct {
	Main       Widget
	Overlays   []Overlay
	Focused    Widget
	Width      int
	Height     int
	GlobalKeys []GlobalKeyBinding
}

func NewRoot(main Widget) *Root {
	return &Root{Main: main}
}

func (r *Root) SetSize(w, h int) {
	r.Width = w
	r.Height = h
	r.Main.SetRect(Rect{X: 0, Y: 0, W: w, H: h})
}

func (r *Root) AddGlobalKey(key tcell.Key, mod tcell.ModMask, rn rune, handler func()) {
	r.GlobalKeys = append(r.GlobalKeys, GlobalKeyBinding{Key: key, Mod: mod, Rune: rn, Handler: handler})
}

func (r *Root) HandleEvent(ev tcell.Event) EventResult {
	// Modal overlay captures all events
	if len(r.Overlays) > 0 {
		top := r.Overlays[len(r.Overlays)-1]
		if top.Modal {
			return top.Widget.HandleEvent(ev)
		}
	}

	// Check global keybindings
	if kev, ok := ev.(*tcell.EventKey); ok {
		for _, gk := range r.GlobalKeys {
			if gk.Key != tcell.KeyRune && kev.Key() == gk.Key && kev.Modifiers() == gk.Mod {
				gk.Handler()
				return EventConsumed
			}
			if gk.Key == tcell.KeyRune && kev.Key() == tcell.KeyRune && kev.Rune() == gk.Rune && kev.Modifiers() == gk.Mod {
				gk.Handler()
				return EventConsumed
			}
		}
	}

	// Route to focused widget
	if r.Focused != nil {
		return r.Focused.HandleEvent(ev)
	}

	return EventIgnored
}

func (r *Root) Render(cells [][]term.Cell) {
	surface := NewRenderSurface(cells, Rect{X: 0, Y: 0, W: r.Width, H: r.Height})
	r.Main.Render(surface)

	for _, overlay := range r.Overlays {
		overlay.Widget.SetRect(Rect{X: 0, Y: 0, W: r.Width, H: r.Height})
		overlay.Widget.Render(surface)
	}
}

func (r *Root) PushOverlay(o Overlay) {
	r.Overlays = append(r.Overlays, o)
}

func (r *Root) PopOverlay() {
	if len(r.Overlays) > 0 {
		r.Overlays = r.Overlays[:len(r.Overlays)-1]
	}
}

func (r *Root) SetFocus(w Widget) {
	r.Focused = w
}
