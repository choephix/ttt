package ui

import (
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type InputAction struct {
	Label   string
	Active  bool
	OnClick func()
}

type InputWidget struct {
	Text         string
	Prefix       string
	Placeholder  string
	CursorPos    int
	scrollOffset int
	Style        term.Style
	Actions      []InputAction
	ActionHits   []HitRegion
	OnChange     func(text string)
}

func NewInputWidget() *InputWidget {
	return &InputWidget{
		Prefix: " ❯ ",
		Style:  term.StyleInput,
	}
}

func (inp *InputWidget) Render(surface *RenderSurface, x, y, w int) {
	actionsW := inp.actionsWidth()
	prefixRunes := []rune(inp.Prefix)
	prefixW := len(prefixRunes)
	textW := w - actionsW - prefixW

	for i, ch := range prefixRunes {
		surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: inp.Style})
	}

	if textW > 0 {
		textRunes := []rune(inp.Text)
		showPlaceholder := len(textRunes) == 0 && inp.Placeholder != ""

		if inp.CursorPos < inp.scrollOffset {
			inp.scrollOffset = inp.CursorPos
		}
		if inp.CursorPos >= inp.scrollOffset+textW {
			inp.scrollOffset = inp.CursorPos - textW + 1
		}

		if showPlaceholder {
			phRunes := []rune(inp.Placeholder)
			for i := 0; i < textW; i++ {
				ch := ' '
				if i < len(phRunes) {
					ch = phRunes[i]
				}
				surface.SetCell(x+prefixW+i, y, term.Cell{Ch: ch, Style: term.StyleInputPlaceholder})
			}
		} else {
			for i := 0; i < textW; i++ {
				ch := ' '
				ri := inp.scrollOffset + i
				if ri < len(textRunes) {
					ch = textRunes[ri]
				}
				surface.SetCell(x+prefixW+i, y, term.Cell{Ch: ch, Style: inp.Style})
			}
		}
	}

	ax := x + prefixW + textW
	inp.ActionHits = inp.ActionHits[:0]
	for _, action := range inp.Actions {
		style := term.StyleInputAction
		if action.Active {
			style = term.StyleDefault
		}
		labelW := len([]rune(action.Label))
		inp.ActionHits = append(inp.ActionHits, HitRegion{X: ax, Y: y, W: labelW})
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
	return x + len([]rune(inp.Prefix)) + inp.CursorPos - inp.scrollOffset
}

func (inp *InputWidget) ResetScroll() {
	inp.scrollOffset = 0
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
	inp.notify()
}

func (inp *InputWidget) Clear() {
	inp.Text = ""
	inp.CursorPos = 0
	inp.notify()
}

func (inp *InputWidget) HandleMouseClick(localX, localY int) bool {
	for i, hit := range inp.ActionHits {
		if localX >= hit.X && localX < hit.X+hit.W && localY == hit.Y {
			if i < len(inp.Actions) && inp.Actions[i].OnClick != nil {
				inp.Actions[i].OnClick()
			}
			return true
		}
	}
	return false
}

func (inp *InputWidget) notify() {
	if inp.OnChange != nil {
		inp.OnChange(inp.Text)
	}
}
