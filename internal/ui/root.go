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

type ChordKeyBinding struct {
	Steps   []GlobalKeyBinding
	Handler func()
}

type chordState struct {
	candidates []int
	stepIdx    int
}

type Root struct {
	Main       Widget
	Overlays   []Overlay
	Focused    Widget
	Width      int
	Height     int
	GlobalKeys []GlobalKeyBinding
	ChordKeys  []ChordKeyBinding
	chord      *chordState
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

func (r *Root) AddChordKey(steps []GlobalKeyBinding, handler func()) {
	r.ChordKeys = append(r.ChordKeys, ChordKeyBinding{Steps: steps, Handler: handler})
}

func matchKey(kev *tcell.EventKey, gk GlobalKeyBinding) bool {
	if gk.Key != tcell.KeyRune {
		return kev.Key() == gk.Key && kev.Modifiers() == gk.Mod
	}
	return kev.Key() == tcell.KeyRune && kev.Rune() == gk.Rune && kev.Modifiers() == gk.Mod
}

func (r *Root) HandleEvent(ev tcell.Event) EventResult {
	if len(r.Overlays) > 0 {
		top := r.Overlays[len(r.Overlays)-1]
		if top.Modal {
			return top.Widget.HandleEvent(ev)
		}
	}

	kev, isKey := ev.(*tcell.EventKey)
	if !isKey {
		if r.Focused != nil {
			return r.Focused.HandleEvent(ev)
		}
		return EventIgnored
	}

	// Mid-chord: try matching next step
	if r.chord != nil {
		var next []int
		for _, ci := range r.chord.candidates {
			chord := r.ChordKeys[ci]
			if r.chord.stepIdx < len(chord.Steps) && matchKey(kev, chord.Steps[r.chord.stepIdx]) {
				if r.chord.stepIdx+1 == len(chord.Steps) {
					r.chord = nil
					chord.Handler()
					return EventConsumed
				}
				next = append(next, ci)
			}
		}
		if len(next) > 0 {
			r.chord.candidates = next
			r.chord.stepIdx++
			return EventConsumed
		}
		r.chord = nil
	}

	// Check if key starts a chord
	var candidates []int
	for i, chord := range r.ChordKeys {
		if len(chord.Steps) > 1 && matchKey(kev, chord.Steps[0]) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) > 0 {
		r.chord = &chordState{candidates: candidates, stepIdx: 1}
		return EventConsumed
	}

	// Single-key global bindings
	for _, gk := range r.GlobalKeys {
		if matchKey(kev, gk) {
			gk.Handler()
			return EventConsumed
		}
	}

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
