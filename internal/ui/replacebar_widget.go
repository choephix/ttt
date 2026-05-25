package ui

import (
	"fmt"
	"github.com/eugenioenko/ttt/internal/term"

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
	focusRow   int
	searchCur  int
	replaceCur int
}

func NewReplaceBarWidget() *ReplaceBarWidget {
	return &ReplaceBarWidget{}
}

func (r *ReplaceBarWidget) Focusable() bool { return true }

func (r *ReplaceBarWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	barW := 40
	if barW > sw-4 {
		barW = sw - 4
	}
	barX := sw - barW - 1
	barY := 4
	barH := 4

	b := term.SingleBorderSet()
	if r.Borders != nil {
		b = *r.Borders
	}
	surface.DrawBorder(barX, barY, barW, barH, b, term.StyleBorder)

	innerW := barW - 2

	// Find row
	findRow := barY + 1
	r.renderRow(surface, barX+1, findRow, innerW, "Find: ", r.Query, r.searchCur, r.focusRow == 0)

	// Replace row
	replRow := barY + 2
	r.renderRow(surface, barX+1, replRow, innerW, "Repl: ", r.Replace, r.replaceCur, r.focusRow == 1)

	// Info + buttons on find row
	info := ""
	if len(r.Query) > 0 {
		info = fmt.Sprintf("%d/%d ", r.currentDisplay(), len(r.Matches))
	}
	buttons := "▲ ▼ ✕"
	suffix := info + buttons
	sx := barX + barW - 2 - len([]rune(suffix))
	if sx > barX {
		surface.DrawText(sx, findRow, suffix, barX+barW-1, term.StyleMuted)
	}

	// Replace buttons on replace row
	replBtns := "⟳ ⟳All"
	rx := barX + barW - 2 - len([]rune(replBtns))
	if rx > barX {
		surface.DrawText(rx, replRow, replBtns, barX+barW-1, term.StyleMuted)
	}
}

func (r *ReplaceBarWidget) renderRow(surface *RenderSurface, startX, y, w int, label, text string, curPos int, focused bool) {
	surface.ClearRect(startX, y, w, 1, term.StyleDefault)
	x := surface.DrawText(startX, y, label, startX+w, term.StyleMuted)
	surface.DrawText(x, y, text, startX+w, term.StyleDefault)

	if focused {
		cx := startX + len([]rune(label)) + curPos
		if cx < startX+w {
			ch := ' '
			runes := []rune(text)
			if curPos < len(runes) {
				ch = runes[curPos]
			}
			surface.SetCell(cx, y, term.Cell{Ch: ch, Style: term.StylePaletteSelected})
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
		return EventConsumed
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
				if kev.Modifiers()&tcell.ModShift != 0 {
					r.Current = (r.Current - 1 + len(r.Matches)) % len(r.Matches)
				} else {
					r.Current = (r.Current + 1) % len(r.Matches)
				}
				r.navigate()
			}
		} else {
			if len(r.Matches) > 0 && r.OnReplace != nil {
				r.OnReplace(r.Matches[r.Current], r.Replace)
				r.search()
			}
		}
		return EventConsumed
	case tcell.KeyUp:
		if len(r.Matches) > 0 {
			r.Current = (r.Current - 1 + len(r.Matches)) % len(r.Matches)
			r.navigate()
		}
		return EventConsumed
	case tcell.KeyDown:
		if len(r.Matches) > 0 {
			r.Current = (r.Current + 1) % len(r.Matches)
			r.navigate()
		}
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
	res := HandleTextEdit(kev, *text, *curPos)
	*text = res.Text
	*curPos = res.CurPos
	if res.Changed && doSearch {
		r.Current = 0
		r.search()
	}
	return EventConsumed
}
