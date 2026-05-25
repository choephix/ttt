package ui

import (
	"github.com/eugenioenko/ttt/internal/term"
	"strconv"

	"github.com/gdamore/tcell/v2"
)

type GoToLineWidget struct {
	BaseWidget
	Input     string
	Borders   *term.BorderSet
	OnSubmit  func(line int)
	OnDismiss func()
}

func NewGoToLineWidget() *GoToLineWidget {
	return &GoToLineWidget{}
}

func (g *GoToLineWidget) Focusable() bool { return true }

func (g *GoToLineWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	boxW := 30
	if boxW > sw-4 {
		boxW = sw - 4
	}
	boxX := (sw - boxW) / 2
	boxY := 2

	b := term.DoubleBorderSet()
	if g.Borders != nil {
		b = *g.Borders
	}

	surface.DrawBorder(boxX, boxY, boxW, 3, b, term.StyleBorder)

	label := "Go to line: "
	inputY := boxY + 1
	x := boxX + 1
	for _, ch := range label {
		if x < boxX+boxW-1 {
			surface.SetCell(x, inputY, term.Cell{Ch: ch, Style: term.StyleDefault})
			x++
		}
	}
	for _, ch := range g.Input {
		if x < boxX+boxW-1 {
			surface.SetCell(x, inputY, term.Cell{Ch: ch, Style: term.StyleDefault})
			x++
		}
	}
	if x < boxX+boxW-1 {
		surface.SetCell(x, inputY, term.Cell{Ch: ' ', Style: term.StyleDefault})
	}
}

func (g *GoToLineWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if g.OnDismiss != nil {
			g.OnDismiss()
		}
		return EventConsumed
	case tcell.KeyEnter:
		if g.OnSubmit != nil {
			n, err := strconv.Atoi(g.Input)
			if err == nil && n > 0 {
				g.OnSubmit(n)
			}
		}
		return EventConsumed
	case tcell.KeyRune:
		if kev.Rune() >= '0' && kev.Rune() <= '9' {
			g.Input += string(kev.Rune())
		}
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.Input) > 0 {
			runes := []rune(g.Input)
			g.Input = string(runes[:len(runes)-1])
		}
		return EventConsumed
	}

	return EventIgnored
}
