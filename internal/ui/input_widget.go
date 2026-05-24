package ui

import (
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type InputAction struct {
	Label   string
	Active  bool
	OnClick func()
}

type InputWidget struct {
	Text      string
	Prefix    string
	CursorPos int
	Style     term.Style
	Actions   []InputAction
	OnChange  func(text string)
}

func NewInputWidget(prefix string) *InputWidget {
	return &InputWidget{
		Prefix: prefix,
		Style:  term.StyleDefault,
	}
}

func (inp *InputWidget) Render(surface *RenderSurface, x, y, w int) {
	actionsW := inp.actionsWidth()
	textW := w - actionsW

	runes := []rune(inp.Prefix + inp.Text)
	for i := 0; i < textW; i++ {
		ch := ' '
		if i < len(runes) {
			ch = runes[i]
		}
		surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: inp.Style})
	}

	ax := x + textW
	for _, action := range inp.Actions {
		style := term.StyleMuted
		if action.Active {
			style = term.StyleDefault
		}
		for _, ch := range action.Label {
			if ax < x+w {
				surface.SetCell(ax, y, term.Cell{Ch: ch, Style: style})
				ax++
			}
		}
		if ax < x+w {
			surface.SetCell(ax, y, term.Cell{Ch: ' ', Style: inp.Style})
			ax++
		}
	}
}

func (inp *InputWidget) actionsWidth() int {
	if len(inp.Actions) == 0 {
		return 0
	}
	w := 0
	for _, a := range inp.Actions {
		w += len([]rune(a.Label)) + 1
	}
	return w
}

func (inp *InputWidget) CursorX(x int) int {
	return x + len([]rune(inp.Prefix)) + inp.CursorPos
}

func (inp *InputWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}
	switch kev.Key() {
	case tcell.KeyRune:
		runes := []rune(inp.Text)
		runes = append(runes[:inp.CursorPos], append([]rune{kev.Rune()}, runes[inp.CursorPos:]...)...)
		inp.Text = string(runes)
		inp.CursorPos++
		inp.notify()
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if inp.CursorPos > 0 {
			runes := []rune(inp.Text)
			runes = append(runes[:inp.CursorPos-1], runes[inp.CursorPos:]...)
			inp.Text = string(runes)
			inp.CursorPos--
			inp.notify()
		}
		return EventConsumed
	case tcell.KeyDelete:
		runes := []rune(inp.Text)
		if inp.CursorPos < len(runes) {
			runes = append(runes[:inp.CursorPos], runes[inp.CursorPos+1:]...)
			inp.Text = string(runes)
			inp.notify()
		}
		return EventConsumed
	case tcell.KeyLeft:
		if inp.CursorPos > 0 {
			inp.CursorPos--
		}
		return EventConsumed
	case tcell.KeyRight:
		if inp.CursorPos < len([]rune(inp.Text)) {
			inp.CursorPos++
		}
		return EventConsumed
	case tcell.KeyHome:
		inp.CursorPos = 0
		return EventConsumed
	case tcell.KeyEnd:
		inp.CursorPos = len([]rune(inp.Text))
		return EventConsumed
	}
	return EventIgnored
}

func (inp *InputWidget) SetText(text string) {
	inp.Text = text
	inp.CursorPos = len([]rune(text))
}

func (inp *InputWidget) Clear() {
	inp.Text = ""
	inp.CursorPos = 0
	inp.notify()
}

func (inp *InputWidget) notify() {
	if inp.OnChange != nil {
		inp.OnChange(inp.Text)
	}
}
