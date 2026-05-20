package ui

import (
	"fmt"
	"ttt/internal/term"
	"strings"

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

func (f *FindBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	surface.Fill(term.Cell{Ch: ' ', Style: term.StylePaletteBorder})

	label := " Find: "
	for i, ch := range label {
		if i < w {
			surface.SetCell(i, 0, term.Cell{Ch: ch, Style: term.StylePaletteBorder})
		}
	}

	inputStart := len([]rune(label))
	queryRunes := []rune(f.Query)
	for i, ch := range queryRunes {
		x := inputStart + i
		if x < w {
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: term.StylePaletteInput})
		}
	}

	cursorX := inputStart + f.cursorPos
	if cursorX < w {
		surface.SetCell(cursorX, 0, term.Cell{Ch: ' ', Style: term.StylePaletteInput})
	}

	if len(f.Query) > 0 {
		info := fmt.Sprintf(" %d/%d ", f.currentDisplay(), len(f.Matches))
		infoStart := w - len([]rune(info))
		if infoStart > 0 {
			for i, ch := range info {
				surface.SetCell(infoStart+i, 0, term.Cell{Ch: ch, Style: term.StylePaletteBorder})
			}
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
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if f.OnDismiss != nil {
			f.OnDismiss()
		}
		return EventConsumed
	case tcell.KeyEnter:
		if len(f.Matches) > 0 {
			f.Current = (f.Current + 1) % len(f.Matches)
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
		return EventIgnored
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
	}

	return EventIgnored
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
