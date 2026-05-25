package ui

import (
	"fmt"
	"strings"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type FindMatch struct {
	Line, Col, Len int
}

type FindBarWidget struct {
	BaseWidget
	Query      string
	Matches    []FindMatch
	Current    int
	Borders    *term.BorderSet
	OnSearch   func(query string) []FindMatch
	OnNavigate func(match FindMatch)
	OnDismiss  func()
	cursorPos  int
}

func NewFindBarWidget() *FindBarWidget {
	return &FindBarWidget{}
}

func (f *FindBarWidget) Focusable() bool { return true }

func (f *FindBarWidget) CursorPosition() (int, int, bool) {
	r := f.GetRect()
	sw := r.W
	barW := 40
	if barW > sw-4 {
		barW = sw - 4
	}
	barX := r.X + sw - barW - 1
	barY := r.Y + 4
	return barX + 2 + f.cursorPos, barY + 1, true
}

func (f *FindBarWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	barW := 40
	if barW > sw-4 {
		barW = sw - 4
	}
	barX := sw - barW - 1
	barY := 4
	barH := 3

	b := term.SingleBorderSet()
	if f.Borders != nil {
		b = *f.Borders
	}
	surface.DrawBorder(barX, barY, barW, barH, b, term.StyleBorder)

	row := barY + 1
	for x := barX + 1; x < barX+barW-1; x++ {
		surface.SetCell(x, row, term.Cell{Ch: ' ', Style: term.StyleDefault})
	}

	x := barX + 2
	queryRunes := []rune(f.Query)
	for i, ch := range queryRunes {
		if x+i < barX+barW-2 {
			surface.SetCell(x+i, row, term.Cell{Ch: ch, Style: term.StyleDefault})
		}
	}

	// Cursor
	cursorX := x + f.cursorPos
	if cursorX < barX+barW-2 {
		ch := ' '
		if f.cursorPos < len(queryRunes) {
			ch = queryRunes[f.cursorPos]
		}
		surface.SetCell(cursorX, row, term.Cell{Ch: ch, Style: term.StylePaletteSelected})
	}

	// Buttons and info on the right
	info := ""
	if len(f.Query) > 0 {
		info = fmt.Sprintf("%d/%d", f.currentDisplay(), len(f.Matches))
	}
	buttons := " ▲ ▼ ✕"
	suffix := info + buttons
	sx := barX + barW - 2 - len([]rune(suffix))
	for i, ch := range suffix {
		if sx+i > barX && sx+i < barX+barW-1 {
			surface.SetCell(sx+i, row, term.Cell{Ch: ch, Style: term.StyleMuted})
		}
	}
}

func (f *FindBarWidget) currentDisplay() int {
	if len(f.Matches) == 0 {
		return 0
	}
	return f.Current + 1
}

func (f *FindBarWidget) search() {
	if f.OnSearch != nil {
		f.Matches = f.OnSearch(f.Query)
		if len(f.Matches) > 0 {
			if f.Current >= len(f.Matches) {
				f.Current = 0
			}
			f.navigate()
		}
	}
}

func (f *FindBarWidget) navigate() {
	if f.OnNavigate != nil && len(f.Matches) > 0 {
		f.OnNavigate(f.Matches[f.Current])
	}
}

func (f *FindBarWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		btn := tev.Buttons()
		mx, my := tev.Position()
		r := f.GetRect()
		sw, _ := r.W, r.H
		barW := 40
		if barW > sw-4 {
			barW = sw - 4
		}
		barX := r.X + sw - barW - 1
		barY := r.Y
		row := barY + 1

		if btn&tcell.Button1 != 0 && my == row {
			buttons := " ▲ ▼ ✕"
			btnStart := barX + barW - 2 - len([]rune(buttons))
			localX := mx - btnStart
			if localX >= 1 && localX <= 1 {
				// ▲ prev
				if len(f.Matches) > 0 {
					f.Current = (f.Current - 1 + len(f.Matches)) % len(f.Matches)
					f.navigate()
				}
				return EventConsumed
			}
			if localX >= 3 && localX <= 3 {
				// ▼ next
				if len(f.Matches) > 0 {
					f.Current = (f.Current + 1) % len(f.Matches)
					f.navigate()
				}
				return EventConsumed
			}
			if localX >= 5 && localX <= 5 {
				// ✕ close
				if f.OnDismiss != nil {
					f.OnDismiss()
				}
				return EventConsumed
			}
		}
		return EventConsumed
	case *tcell.EventKey:
		return f.handleKey(tev)
	}
	return EventConsumed
}

func (f *FindBarWidget) handleKey(kev *tcell.EventKey) EventResult {
	switch kev.Key() {
	case tcell.KeyEscape:
		if f.OnDismiss != nil {
			f.OnDismiss()
		}
		return EventConsumed
	case tcell.KeyEnter:
		if len(f.Matches) > 0 {
			if kev.Modifiers()&tcell.ModShift != 0 {
				f.Current = (f.Current - 1 + len(f.Matches)) % len(f.Matches)
			} else {
				f.Current = (f.Current + 1) % len(f.Matches)
			}
			f.navigate()
		}
		return EventConsumed
	case tcell.KeyRune:
		if kev.Modifiers() == 0 {
			runes := []rune(f.Query)
			runes = append(runes[:f.cursorPos], append([]rune{kev.Rune()}, runes[f.cursorPos:]...)...)
			f.Query = string(runes)
			f.cursorPos++
			f.Current = 0
			f.search()
			return EventConsumed
		}
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if f.cursorPos > 0 {
			runes := []rune(f.Query)
			runes = append(runes[:f.cursorPos-1], runes[f.cursorPos:]...)
			f.Query = string(runes)
			f.cursorPos--
			f.Current = 0
			f.search()
		}
		return EventConsumed
	case tcell.KeyDelete:
		runes := []rune(f.Query)
		if f.cursorPos < len(runes) {
			runes = append(runes[:f.cursorPos], runes[f.cursorPos+1:]...)
			f.Query = string(runes)
			f.Current = 0
			f.search()
		}
		return EventConsumed
	case tcell.KeyLeft:
		if f.cursorPos > 0 {
			f.cursorPos--
		}
		return EventConsumed
	case tcell.KeyRight:
		if f.cursorPos < len([]rune(f.Query)) {
			f.cursorPos++
		}
		return EventConsumed
	case tcell.KeyHome:
		f.cursorPos = 0
		return EventConsumed
	case tcell.KeyEnd:
		f.cursorPos = len([]rune(f.Query))
		return EventConsumed
	case tcell.KeyUp:
		if len(f.Matches) > 0 {
			f.Current = (f.Current - 1 + len(f.Matches)) % len(f.Matches)
			f.navigate()
		}
		return EventConsumed
	case tcell.KeyDown:
		if len(f.Matches) > 0 {
			f.Current = (f.Current + 1) % len(f.Matches)
			f.navigate()
		}
		return EventConsumed
	}

	return EventConsumed
}

func FindInLines(lines []string, query string) []FindMatch {
	if query == "" {
		return nil
	}
	lowerQuery := strings.ToLower(query)
	queryLen := len([]rune(lowerQuery))
	var matches []FindMatch
	for lineIdx, line := range lines {
		lowerLine := strings.ToLower(line)
		offset := 0
		for {
			idx := strings.Index(lowerLine[offset:], lowerQuery)
			if idx < 0 {
				break
			}
			col := len([]rune(lowerLine[:offset+idx]))
			matches = append(matches, FindMatch{Line: lineIdx, Col: col, Len: queryLen})
			offset += idx + len(lowerQuery)
		}
	}
	return matches
}
