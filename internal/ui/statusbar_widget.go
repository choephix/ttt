package ui

import (
	"fmt"
	"ttt/internal/term"
	"ttt/internal/view"

	"github.com/gdamore/tcell/v2"
)

type statusBarSpan struct {
	start, end int
}

type StatusBarWidget struct {
	BaseWidget
	Status         *view.StatusBar
	OnIndentClick  func()
	indentSpan     statusBarSpan
}

func NewStatusBarWidget(status *view.StatusBar) *StatusBarWidget {
	return &StatusBarWidget{Status: status}
}

func (s *StatusBarWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	st := s.Status

	for x := 0; x < w; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: term.StyleStatusBar})
	}

	x := 0
	x += s.drawText(surface, x, " ", term.StyleStatusBar)

	if st.Branch != "" {
		x += s.drawText(surface, x, st.Branch, term.StyleStatusBar)
		x += s.drawText(surface, x, "  ", term.StyleStatusBar)
	}

	if st.Blame != "" {
		x += s.drawText(surface, x, st.Blame, term.StyleStatusBarMuted)
	}

	type segment struct {
		text string
		id   string
	}
	var right []segment
	right = append(right, segment{fmt.Sprintf("Ln %d, Col %d", st.Line+1, st.Col+1), "pos"})
	if st.TabSize > 0 {
		right = append(right, segment{fmt.Sprintf("Spaces: %d", st.TabSize), "indent"})
	}
	right = append(right, segment{"UTF-8", "encoding"})
	right = append(right, segment{"LF", "eol"})
	if st.Language != "" {
		right = append(right, segment{st.Language, "lang"})
	}

	rightStr := ""
	for i, seg := range right {
		if i > 0 {
			rightStr += "   "
		}
		rightStr += seg.text
	}
	rightStr += " "

	rx := w - len([]rune(rightStr))
	if rx > x {
		pos := rx
		r := s.GetRect()
		for i, seg := range right {
			if i > 0 {
				s.drawText(surface, pos, "   ", term.StyleStatusBar)
				pos += 3
			}
			segLen := len([]rune(seg.text))
			if seg.id == "indent" {
				s.indentSpan = statusBarSpan{r.X + pos, r.X + pos + segLen}
			}
			s.drawText(surface, pos, seg.text, term.StyleStatusBar)
			pos += segLen
		}
	}
}

func (s *StatusBarWidget) HandleEvent(ev tcell.Event) EventResult {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return EventIgnored
	}
	if mev.Buttons()&tcell.Button1 == 0 {
		return EventIgnored
	}
	mx, my := mev.Position()
	r := s.GetRect()
	if my != r.Y {
		return EventIgnored
	}
	if mx >= s.indentSpan.start && mx < s.indentSpan.end && s.OnIndentClick != nil {
		s.OnIndentClick()
		return EventConsumed
	}
	return EventIgnored
}

func (s *StatusBarWidget) drawText(surface *RenderSurface, x int, text string, style term.Style) int {
	w, _ := surface.Size()
	n := 0
	for _, ch := range text {
		if x+n >= w {
			break
		}
		surface.SetCell(x+n, 0, term.Cell{Ch: ch, Style: style})
		n++
	}
	return n
}
