package ui

import (
	"fmt"
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
	Options    SearchOptions
	Borders    *term.BorderSet
	OnSearch   func(query string, opts SearchOptions) []FindMatch
	OnNavigate func(match FindMatch)
	OnDismiss  func()
	cursorPos  int
	btnCase    HitRegion
	btnRegex   HitRegion
	btnPrev    HitRegion
	btnNext    HitRegion
	btnClose   HitRegion
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
	surface.ClearRect(barX+1, row, barW-2, 1, term.StyleDefault)

	x := barX + 2
	queryRunes := []rune(f.Query)
	surface.DrawText(x, row, f.Query, barX+barW-2, term.StyleDefault)

	// Cursor
	cursorX := x + f.cursorPos
	if cursorX < barX+barW-2 {
		ch := ' '
		if f.cursorPos < len(queryRunes) {
			ch = queryRunes[f.cursorPos]
		}
		surface.SetCell(cursorX, row, term.Cell{Ch: ch, Style: term.StylePaletteSelected})
	}

	// Toggle buttons + info + nav on the right
	type toggle struct {
		label  string
		active bool
	}
	toggles := []toggle{
		{"Aa", f.Options.CaseSensitive},
		{".*", f.Options.UseRegex},
	}

	info := ""
	if len(f.Query) > 0 {
		info = fmt.Sprintf("%d/%d", f.currentDisplay(), len(f.Matches))
	}

	toggleW := 0
	for _, tg := range toggles {
		toggleW += len([]rune(tg.label)) + 1
	}
	navButtons := " ▲ ▼ ✕"
	totalW := toggleW + len([]rune(info)) + len([]rune(navButtons))
	sx := barX + barW - 2 - totalW
	if sx <= barX {
		sx = barX + 1
	}

	cx := sx
	toggleHits := []*HitRegion{&f.btnCase, &f.btnRegex}
	for i, tg := range toggles {
		style := term.StyleMuted
		if tg.active {
			style = term.StyleDefault
		}
		*toggleHits[i] = HitRegion{X: cx, Y: row, W: len([]rune(tg.label))}
		for _, ch := range tg.label {
			if cx < barX+barW-1 {
				surface.SetCell(cx, row, term.Cell{Ch: ch, Style: style})
				cx++
			}
		}
		if cx < barX+barW-1 {
			surface.SetCell(cx, row, term.Cell{Ch: ' ', Style: term.StyleMuted})
			cx++
		}
	}

	for _, ch := range info {
		if cx < barX+barW-1 {
			surface.SetCell(cx, row, term.Cell{Ch: ch, Style: term.StyleMuted})
			cx++
		}
	}

	navStart := cx
	for _, ch := range navButtons {
		if cx < barX+barW-1 {
			surface.SetCell(cx, row, term.Cell{Ch: ch, Style: term.StyleMuted})
			cx++
		}
	}
	f.btnPrev = HitRegion{X: navStart + 1, Y: row, W: 1}
	f.btnNext = HitRegion{X: navStart + 3, Y: row, W: 1}
	f.btnClose = HitRegion{X: navStart + 5, Y: row, W: 1}
}

func (f *FindBarWidget) currentDisplay() int {
	if len(f.Matches) == 0 {
		return 0
	}
	return f.Current + 1
}

func (f *FindBarWidget) search() {
	if f.OnSearch != nil {
		f.Matches = f.OnSearch(f.Query, f.Options)
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
		if tev.Buttons()&tcell.Button1 != 0 {
			mx, my := tev.Position()
			if f.btnCase.Contains(mx, my) {
				f.Options.CaseSensitive = !f.Options.CaseSensitive
				f.Current = 0
				f.search()
				return EventConsumed
			}
			if f.btnRegex.Contains(mx, my) {
				f.Options.UseRegex = !f.Options.UseRegex
				f.Current = 0
				f.search()
				return EventConsumed
			}
			if f.btnPrev.Contains(mx, my) {
				if len(f.Matches) > 0 {
					f.Current = (f.Current - 1 + len(f.Matches)) % len(f.Matches)
					f.navigate()
				}
				return EventConsumed
			}
			if f.btnNext.Contains(mx, my) {
				if len(f.Matches) > 0 {
					f.Current = (f.Current + 1) % len(f.Matches)
					f.navigate()
				}
				return EventConsumed
			}
			if f.btnClose.Contains(mx, my) {
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
		if kev.Modifiers()&tcell.ModAlt != 0 {
			switch kev.Rune() {
			case 'c':
				f.Options.CaseSensitive = !f.Options.CaseSensitive
				f.Current = 0
				f.search()
				return EventConsumed
			case 'r':
				f.Options.UseRegex = !f.Options.UseRegex
				f.Current = 0
				f.search()
				return EventConsumed
			}
		}
		r := HandleTextEdit(kev, f.Query, f.cursorPos)
		f.Query = r.Text
		f.cursorPos = r.CurPos
		if r.Changed {
			f.Current = 0
			f.search()
		}
		return EventConsumed
	case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete,
		tcell.KeyLeft, tcell.KeyRight, tcell.KeyHome, tcell.KeyEnd:
		r := HandleTextEdit(kev, f.Query, f.cursorPos)
		f.Query = r.Text
		f.cursorPos = r.CurPos
		if r.Changed {
			f.Current = 0
			f.search()
		}
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

