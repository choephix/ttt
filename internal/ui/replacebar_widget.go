package ui

import (
	"fmt"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ReplaceBarWidget struct {
	BaseWidget
	Query      string
	Replace    string
	Matches    []FindMatch
	Current    int
	Borders    *term.BorderSet
	OnSearch   func(query string) []FindMatch
	OnNavigate func(match FindMatch)
	OnReplace  func(match FindMatch, replacement string)
	OnReplaceAll func(query, replacement string)
	OnDismiss  func()
	focusRow   int // 0 = search, 1 = replace
	searchCur  int
	replaceCur int
}

func NewReplaceBarWidget() *ReplaceBarWidget {
	return &ReplaceBarWidget{}
}

func (r *ReplaceBarWidget) Focusable() bool { return true }

func (r *ReplaceBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	surface.Fill(term.Cell{Ch: ' ', Style: term.StylePaletteBorder})

	findLabel := " Find:    "
	replLabel := " Replace: "

	r.renderRow(surface, 0, w, findLabel, r.Query, r.searchCur, r.focusRow == 0)
	r.renderRow(surface, 1, w, replLabel, r.Replace, r.replaceCur, r.focusRow == 1)

	if len(r.Query) > 0 {
		info := fmt.Sprintf(" %d/%d ", r.currentDisplay(), len(r.Matches))
		infoStart := w - len([]rune(info))
		if infoStart > 0 {
			for i, ch := range info {
				surface.SetCell(infoStart+i, 0, term.Cell{Ch: ch, Style: term.StylePaletteBorder})
			}
		}
	}
}

func (r *ReplaceBarWidget) renderRow(surface *RenderSurface, row, w int, label, text string, curPos int, focused bool) {
	for i, ch := range label {
		if i < w {
			surface.SetCell(i, row, term.Cell{Ch: ch, Style: term.StylePaletteBorder})
		}
	}
	inputStart := len([]rune(label))
	for i, ch := range []rune(text) {
		x := inputStart + i
		if x < w {
			surface.SetCell(x, row, term.Cell{Ch: ch, Style: term.StylePaletteInput})
		}
	}
	if focused {
		cx := inputStart + curPos
		if cx < w {
			surface.SetCell(cx, row, term.Cell{Ch: ' ', Style: term.StylePaletteInput})
		}
	}
}

func (r *ReplaceBarWidget) currentDisplay() int {
	if len(r.Matches) == 0 {
		return 0
	}
	return r.Current + 1
}

func (r *ReplaceBarWidget) search() {
	if r.OnSearch != nil {
		r.Matches = r.OnSearch(r.Query)
		if len(r.Matches) > 0 {
			if r.Current >= len(r.Matches) {
				r.Current = 0
			}
			r.navigate()
		}
	}
}

func (r *ReplaceBarWidget) navigate() {
	if r.OnNavigate != nil && len(r.Matches) > 0 {
		r.OnNavigate(r.Matches[r.Current])
	}
}

func (r *ReplaceBarWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if r.OnDismiss != nil {
			r.OnDismiss()
		}
		return EventConsumed
	case tcell.KeyTab, tcell.KeyBacktab:
		r.focusRow = 1 - r.focusRow
		return EventConsumed
	case tcell.KeyEnter:
		if r.focusRow == 0 {
			if len(r.Matches) > 0 {
				r.Current = (r.Current + 1) % len(r.Matches)
				r.navigate()
			}
		} else {
			if len(r.Matches) > 0 && r.OnReplace != nil {
				r.OnReplace(r.Matches[r.Current], r.Replace)
				r.search()
			}
		}
		return EventConsumed
	case tcell.KeyCtrlA + 7: // Ctrl+H — replace all shortcut within the bar
		return EventConsumed
	}

	if kev.Modifiers()&tcell.ModCtrl != 0 && kev.Modifiers()&tcell.ModShift != 0 {
		return EventConsumed
	}

	if kev.Modifiers()&tcell.ModAlt != 0 && kev.Key() == tcell.KeyRune && kev.Rune() == 'r' {
		if r.OnReplaceAll != nil {
			r.OnReplaceAll(r.Query, r.Replace)
			r.search()
		}
		return EventConsumed
	}

	if r.focusRow == 0 {
		return r.handleInput(kev, &r.Query, &r.searchCur, true)
	}
	return r.handleInput(kev, &r.Replace, &r.replaceCur, false)
}

func (r *ReplaceBarWidget) handleInput(kev *tcell.EventKey, text *string, curPos *int, doSearch bool) EventResult {
	switch kev.Key() {
	case tcell.KeyRune:
		if kev.Modifiers() != 0 {
			return EventIgnored
		}
		runes := []rune(*text)
		runes = append(runes[:*curPos], append([]rune{kev.Rune()}, runes[*curPos:]...)...)
		*text = string(runes)
		*curPos++
		if doSearch {
			r.Current = 0
			r.search()
		}
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if *curPos > 0 {
			runes := []rune(*text)
			runes = append(runes[:*curPos-1], runes[*curPos:]...)
			*text = string(runes)
			*curPos--
			if doSearch {
				r.Current = 0
				r.search()
			}
		}
		return EventConsumed
	case tcell.KeyDelete:
		runes := []rune(*text)
		if *curPos < len(runes) {
			runes = append(runes[:*curPos], runes[*curPos+1:]...)
			*text = string(runes)
			if doSearch {
				r.Current = 0
				r.search()
			}
		}
		return EventConsumed
	case tcell.KeyLeft:
		if *curPos > 0 {
			*curPos--
		}
		return EventConsumed
	case tcell.KeyRight:
		if *curPos < len([]rune(*text)) {
			*curPos++
		}
		return EventConsumed
	case tcell.KeyHome:
		*curPos = 0
		return EventConsumed
	case tcell.KeyEnd:
		*curPos = len([]rune(*text))
		return EventConsumed
	}
	return EventIgnored
}
