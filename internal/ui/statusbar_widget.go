package ui

import (
	"fmt"
	"ttt/internal/term"
	"ttt/internal/view"
)

type StatusBarWidget struct {
	BaseWidget
	Status *view.StatusBar
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

	var right []string
	right = append(right, fmt.Sprintf("Ln %d, Col %d", st.Line+1, st.Col+1))
	if st.TabSize > 0 {
		right = append(right, fmt.Sprintf("Spaces: %d", st.TabSize))
	}
	right = append(right, "UTF-8")
	right = append(right, "LF")
	if st.Language != "" {
		right = append(right, st.Language)
	}

	rightStr := ""
	for i, seg := range right {
		if i > 0 {
			rightStr += "   "
		}
		rightStr += seg
	}
	rightStr += " "

	rx := w - len([]rune(rightStr))
	if rx > x {
		s.drawText(surface, rx, rightStr, term.StyleStatusBar)
	}
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
