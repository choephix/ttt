package widgets

import (
	"strings"
	"time"
	"unicode"

	"github.com/eugenioenko/ttt/internal/core/clipboard"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type InputConfig struct {
	Prefix      string `json:"prefix,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Bordered    bool   `json:"bordered"`
	Style       term.Style `json:"-"`

	OnChange func(text string)
	OnSubmit func(text string)
}

type InputWidget struct {
	BaseWidget
	Config        InputConfig
	text          string
	cursorPos     int
	scrollOffset  int
	selStart      int
	selEnd        int
	focused       bool
	lastClickTime int64
	lastClickPos  int
	clickCount    int
}

func NewInputWidget(config InputConfig) *InputWidget {
	if !config.Bordered && config.Prefix == "" {
		config.Prefix = " ❯ "
	}
	return &InputWidget{
		Config:   config,
		selStart: -1,
	}
}

func (inp *InputWidget) Height() int {
	h := 1 + inp.BoxOverheadH()
	if inp.Config.Bordered {
		h += 2
	}
	return h
}

func (inp *InputWidget) Width() int  { return 0 }

func (inp *InputWidget) Focusable() bool    { return true }
func (inp *InputWidget) SetFocused(f bool)  { inp.focused = f }
func (inp *InputWidget) IsFocused() bool    { return inp.focused }

func (inp *InputWidget) CursorPosition() (int, int, bool) {
	if !inp.focused {
		return 0, 0, false
	}
	r := inp.GetRect()
	textX := r.X + inp.Box.MarginLeft + inp.Box.PaddingLeft
	textY := r.Y + inp.Box.MarginTop + inp.Box.PaddingTop
	if inp.Config.Bordered {
		textX += 2
		textY += 1
	} else {
		textX += len([]rune(inp.Config.Prefix))
	}
	return textX + inp.cursorPos - inp.scrollOffset, textY, true
}

func (inp *InputWidget) Text() string     { return inp.text }
func (inp *InputWidget) SetText(t string) {
	inp.text = t
	inp.cursorPos = len([]rune(t))
	inp.clearSel()
	inp.notify()
}

func (inp *InputWidget) ResetScroll() {
	inp.scrollOffset = 0
}

func (inp *InputWidget) PasteText(text string) {
	if inp.hasSelection() {
		inp.deleteSelection()
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimRight(text, "\n")
	text = strings.ReplaceAll(text, "\n", " ")
	if text == "" {
		return
	}
	runes := []rune(inp.text)
	pasted := []rune(text)
	runes = append(runes[:inp.cursorPos], append(pasted, runes[inp.cursorPos:]...)...)
	inp.text = string(runes)
	inp.cursorPos += len(pasted)
	inp.notify()
}

func (inp *InputWidget) Clear() {
	inp.text = ""
	inp.cursorPos = 0
	inp.scrollOffset = 0
	inp.clearSel()
	inp.notify()
}

func (inp *InputWidget) Render(surface Surface) {
	if inp.Config.Bordered {
		inp.renderBordered(surface)
	} else {
		inp.renderBorderless(surface)
	}
}

func (inp *InputWidget) renderBordered(surface Surface) {
	inner := inp.RenderBox(surface)
	w, h := inner.Size()
	if w < 3 || h < 3 {
		return
	}

	borderStyle := term.StyleBorder
	if inp.focused {
		borderStyle = term.StyleBorderActive
	}
	bs := inp.borders()

	inner.DrawBorder(0, 0, w, h, bs, borderStyle)

	inp.renderText(inner, 2, 1, w-4)
}

func (inp *InputWidget) borders() term.BorderSet {
	if inp.Box.Borders.Horizontal != 0 {
		return inp.Box.Borders
	}
	return term.BorderSet{
		Horizontal: '─', Vertical: '│',
		TopLeft: '╭', TopRight: '╮',
		BottomLeft: '╰', BottomRight: '╯',
	}
}

func (inp *InputWidget) renderBorderless(surface Surface) {
	inner := inp.RenderBox(surface)
	w, _ := inner.Size()
	if w <= 0 {
		return
	}

	prefixRunes := []rune(inp.Config.Prefix)
	prefixW := len(prefixRunes)

	prefixStyle := term.StyleBorder
	if inp.focused {
		prefixStyle = term.StyleBorderActive
	}

	for i, ch := range prefixRunes {
		if i < w {
			inner.SetCell(i, 0, term.Cell{Ch: ch, Style: prefixStyle})
		}
	}

	inp.renderText(inner, prefixW, 0, w-prefixW)
}

func (inp *InputWidget) renderText(surface Surface, x, y, textW int) {
	if textW <= 0 {
		return
	}

	style := inp.Config.Style
	if style == 0 {
		style = term.StyleInput
	}

	textRunes := []rune(inp.text)

	if inp.cursorPos < inp.scrollOffset {
		inp.scrollOffset = inp.cursorPos
	}
	if inp.cursorPos >= inp.scrollOffset+textW {
		inp.scrollOffset = inp.cursorPos - textW + 1
	}

	if len(textRunes) == 0 && inp.Config.Placeholder != "" {
		phRunes := []rune(inp.Config.Placeholder)
		for i := range textW {
			ch := ' '
			if i < len(phRunes) {
				ch = phRunes[i]
			}
			surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: term.StyleInputPlaceholder})
		}
		return
	}

	selLo, selHi := -1, -1
	if inp.hasSelection() {
		selLo, selHi = inp.selRange()
	}

	for i := range textW {
		ch := ' '
		ri := inp.scrollOffset + i
		if ri < len(textRunes) {
			ch = textRunes[ri]
		}
		s := style
		if selLo >= 0 && ri >= selLo && ri < selHi {
			s = term.StyleSelection
		}
		surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: s})
	}
}

func (inp *InputWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return inp.handleKey(tev)
	case *tcell.EventMouse:
		return inp.handleMouse(tev)
	}
	return EventIgnored
}

func (inp *InputWidget) handleKey(ev *tcell.EventKey) EventResult {
	if !inp.focused {
		return EventIgnored
	}
	shift := ev.Modifiers()&tcell.ModShift != 0
	ctrl := ev.Modifiers()&tcell.ModCtrl != 0

	switch ev.Key() {
	case tcell.KeyRune:
		if inp.hasSelection() {
			inp.deleteSelection()
		}
		runes := []rune(inp.text)
		runes = append(runes[:inp.cursorPos], append([]rune{ev.Rune()}, runes[inp.cursorPos:]...)...)
		inp.text = string(runes)
		inp.cursorPos++
		inp.notify()
		return EventConsumed
	case tcell.KeyEnter:
		if !shift && inp.Config.OnSubmit != nil {
			inp.Config.OnSubmit(inp.text)
		}
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if inp.hasSelection() {
			inp.deleteSelection()
		} else if inp.cursorPos > 0 {
			runes := []rune(inp.text)
			newPos := inp.cursorPos - 1
			if ctrl {
				newPos = inp.wordLeft()
			}
			inp.text = string(append(runes[:newPos], runes[inp.cursorPos:]...))
			inp.cursorPos = newPos
			inp.notify()
		}
		return EventConsumed
	case tcell.KeyDelete:
		if inp.hasSelection() {
			inp.deleteSelection()
		} else {
			runes := []rune(inp.text)
			if inp.cursorPos < len(runes) {
				end := inp.cursorPos + 1
				if ctrl {
					end = inp.wordRight()
				}
				inp.text = string(append(runes[:inp.cursorPos], runes[end:]...))
				inp.notify()
			}
		}
		return EventConsumed
	case tcell.KeyLeft:
		if shift {
			inp.startSel()
		}
		if !shift && inp.hasSelection() {
			lo, _ := inp.selRange()
			inp.cursorPos = lo
			inp.clearSel()
		} else if ctrl {
			inp.cursorPos = inp.wordLeft()
		} else if inp.cursorPos > 0 {
			inp.cursorPos--
		}
		if shift {
			inp.selEnd = inp.cursorPos
		}
		return EventConsumed
	case tcell.KeyRight:
		if shift {
			inp.startSel()
		}
		if !shift && inp.hasSelection() {
			_, hi := inp.selRange()
			inp.cursorPos = hi
			inp.clearSel()
		} else if ctrl {
			inp.cursorPos = inp.wordRight()
		} else if inp.cursorPos < len([]rune(inp.text)) {
			inp.cursorPos++
		}
		if shift {
			inp.selEnd = inp.cursorPos
		}
		return EventConsumed
	case tcell.KeyHome:
		if shift {
			inp.startSel()
		}
		inp.cursorPos = 0
		if shift {
			inp.selEnd = inp.cursorPos
		} else {
			inp.clearSel()
		}
		return EventConsumed
	case tcell.KeyEnd:
		if shift {
			inp.startSel()
		}
		inp.cursorPos = len([]rune(inp.text))
		if shift {
			inp.selEnd = inp.cursorPos
		} else {
			inp.clearSel()
		}
		return EventConsumed
	case tcell.KeyCtrlV:
		inp.pasteClipboard()
		return EventConsumed
	case tcell.KeyCtrlA:
		inp.selectAll()
		return EventConsumed
	case tcell.KeyCtrlC:
		inp.copySelection()
		return EventConsumed
	case tcell.KeyCtrlX:
		inp.cutSelection()
		return EventConsumed
	}
	return EventIgnored
}

func (inp *InputWidget) handleMouse(ev *tcell.EventMouse) EventResult {
	if ev.Buttons()&tcell.Button1 == 0 {
		return EventIgnored
	}
	mx, my := ev.Position()
	r := inp.GetRect()
	if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
		return EventIgnored
	}

	textX := r.X + inp.Box.MarginLeft + inp.Box.PaddingLeft + len([]rune(inp.Config.Prefix))
	if inp.Config.Bordered {
		textX = r.X + inp.Box.MarginLeft + inp.Box.PaddingLeft + 2
	}
	pos := inp.scrollOffset + (mx - textX)
	runes := []rune(inp.text)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}

	now := time.Now().UnixMilli()
	if now-inp.lastClickTime < 400 && pos == inp.lastClickPos {
		inp.clickCount++
	} else {
		inp.clickCount = 1
	}
	inp.lastClickTime = now
	inp.lastClickPos = pos

	if inp.clickCount == 2 {
		inp.selectWordAt(pos)
	} else {
		inp.cursorPos = pos
		inp.clearSel()
	}
	return EventConsumed
}

// selection helpers

func (inp *InputWidget) hasSelection() bool {
	return inp.selStart >= 0 && inp.selStart != inp.selEnd
}

func (inp *InputWidget) selRange() (int, int) {
	if inp.selStart < inp.selEnd {
		return inp.selStart, inp.selEnd
	}
	return inp.selEnd, inp.selStart
}

func (inp *InputWidget) clearSel() {
	inp.selStart = -1
	inp.selEnd = -1
}

func (inp *InputWidget) startSel() {
	if inp.selStart < 0 {
		inp.selStart = inp.cursorPos
		inp.selEnd = inp.cursorPos
	}
}

func (inp *InputWidget) deleteSelection() {
	lo, hi := inp.selRange()
	runes := []rune(inp.text)
	if lo > len(runes) {
		lo = len(runes)
	}
	if hi > len(runes) {
		hi = len(runes)
	}
	inp.text = string(append(runes[:lo], runes[hi:]...))
	inp.cursorPos = lo
	inp.clearSel()
	inp.notify()
}

func (inp *InputWidget) selectWordAt(pos int) {
	runes := []rune(inp.text)
	if pos < 0 || pos >= len(runes) {
		return
	}
	lo, hi := pos, pos
	if isInputWordRune(runes[pos]) {
		for lo > 0 && isInputWordRune(runes[lo-1]) {
			lo--
		}
		for hi < len(runes) && isInputWordRune(runes[hi]) {
			hi++
		}
	} else if !unicode.IsSpace(runes[pos]) {
		for lo > 0 && !isInputWordRune(runes[lo-1]) && !unicode.IsSpace(runes[lo-1]) {
			lo--
		}
		for hi < len(runes) && !isInputWordRune(runes[hi]) && !unicode.IsSpace(runes[hi]) {
			hi++
		}
	} else {
		for lo > 0 && unicode.IsSpace(runes[lo-1]) {
			lo--
		}
		for hi < len(runes) && unicode.IsSpace(runes[hi]) {
			hi++
		}
	}
	inp.selStart = lo
	inp.selEnd = hi
	inp.cursorPos = hi
}

func (inp *InputWidget) selectAll() {
	runes := []rune(inp.text)
	if len(runes) == 0 {
		return
	}
	inp.selStart = 0
	inp.selEnd = len(runes)
	inp.cursorPos = len(runes)
}

func (inp *InputWidget) copySelection() {
	if !inp.hasSelection() {
		return
	}
	lo, hi := inp.selRange()
	runes := []rune(inp.text)
	clipboard.Set(string(runes[lo:hi]))
}

func (inp *InputWidget) cutSelection() {
	if !inp.hasSelection() {
		return
	}
	inp.copySelection()
	inp.deleteSelection()
}

func (inp *InputWidget) pasteClipboard() {
	inp.PasteText(clipboard.Get())
}

// word navigation

func (inp *InputWidget) wordLeft() int {
	runes := []rune(inp.text)
	pos := inp.cursorPos - 1
	if pos >= len(runes) {
		pos = len(runes) - 1
	}
	if pos < 0 {
		return 0
	}
	if unicode.IsSpace(runes[pos]) {
		for pos > 0 && unicode.IsSpace(runes[pos-1]) {
			pos--
		}
	} else if isInputWordRune(runes[pos]) {
		for pos > 0 && isInputWordRune(runes[pos-1]) {
			pos--
		}
	} else {
		for pos > 0 && !isInputWordRune(runes[pos-1]) && !unicode.IsSpace(runes[pos-1]) {
			pos--
		}
	}
	return pos
}

func (inp *InputWidget) wordRight() int {
	runes := []rune(inp.text)
	pos := inp.cursorPos
	if pos >= len(runes) {
		return len(runes)
	}
	if unicode.IsSpace(runes[pos]) {
		for pos < len(runes) && unicode.IsSpace(runes[pos]) {
			pos++
		}
	} else if isInputWordRune(runes[pos]) {
		for pos < len(runes) && isInputWordRune(runes[pos]) {
			pos++
		}
	} else {
		for pos < len(runes) && !isInputWordRune(runes[pos]) && !unicode.IsSpace(runes[pos]) {
			pos++
		}
	}
	return pos
}

func isInputWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (inp *InputWidget) notify() {
	if inp.Config.OnChange != nil {
		inp.Config.OnChange(inp.text)
	}
}
