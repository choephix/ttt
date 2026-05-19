package ui

import (
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type SplitPanelWidget struct {
	BaseWidget
	Left       Widget
	Right      Widget
	LeftTitle  string
	RightTitle string
	DividerPos int
	Borders    *term.BorderSet
	ShowLeft   bool
}

func NewSplitPanelWidget() *SplitPanelWidget {
	return &SplitPanelWidget{
		DividerPos: 30,
		ShowLeft:   true,
	}
}

func (s *SplitPanelWidget) Focusable() bool { return false }

func (s *SplitPanelWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	if w < 4 || h < 3 {
		return
	}

	b := term.SingleBorderSet()
	if s.Borders != nil {
		b = *s.Borders
	}
	bs := term.StyleBorder
	r := s.GetRect()

	if !s.ShowLeft {
		s.renderSinglePanel(surface, w, h, b, bs)
		return
	}

	divX := s.DividerPos + 1
	if divX < 2 {
		divX = 2
	}
	if divX >= w-2 {
		divX = w - 3
	}

	// Top border
	surface.SetCell(0, 0, term.Cell{Ch: b.TopLeft, Style: bs})
	for x := 1; x < divX; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(divX, 0, term.Cell{Ch: b.TopTee, Style: bs})
	for x := divX + 1; x < w-1; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(w-1, 0, term.Cell{Ch: b.TopRight, Style: bs})

	// Bottom border
	surface.SetCell(0, h-1, term.Cell{Ch: b.BottomLeft, Style: bs})
	for x := 1; x < divX; x++ {
		surface.SetCell(x, h-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(divX, h-1, term.Cell{Ch: b.BottomTee, Style: bs})
	for x := divX + 1; x < w-1; x++ {
		surface.SetCell(x, h-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(w-1, h-1, term.Cell{Ch: b.BottomRight, Style: bs})

	// Left border, divider, right border
	for y := 1; y < h-1; y++ {
		surface.SetCell(0, y, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(divX, y, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(w-1, y, term.Cell{Ch: b.Vertical, Style: bs})
	}

	// Left title in top border
	s.drawTitle(surface, 1, 0, divX-1, s.LeftTitle, b, bs)

	// Right title in top border
	s.drawTitle(surface, divX+1, 0, w-1-divX-1, s.RightTitle, b, bs)

	// Render left content (absolute coords for SetRect, local for Sub)
	if s.Left != nil {
		leftW := divX - 1
		leftH := h - 2
		if leftW > 0 && leftH > 0 {
			s.Left.SetRect(Rect{X: r.X + 1, Y: r.Y + 1, W: leftW, H: leftH})
			leftSurface := surface.Sub(Rect{X: 1, Y: 1, W: leftW, H: leftH})
			s.Left.Render(leftSurface)
		}
	}

	// Render right content (absolute coords for SetRect, local for Sub)
	if s.Right != nil {
		rightW := w - divX - 2
		rightH := h - 2
		if rightW > 0 && rightH > 0 {
			s.Right.SetRect(Rect{X: r.X + divX + 1, Y: r.Y + 1, W: rightW, H: rightH})
			rightSurface := surface.Sub(Rect{X: divX + 1, Y: 1, W: rightW, H: rightH})
			s.Right.Render(rightSurface)
		}
	}
}

func (s *SplitPanelWidget) renderSinglePanel(surface *RenderSurface, w, h int, b term.BorderSet, bs term.Style) {
	// Top border
	surface.SetCell(0, 0, term.Cell{Ch: b.TopLeft, Style: bs})
	for x := 1; x < w-1; x++ {
		surface.SetCell(x, 0, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(w-1, 0, term.Cell{Ch: b.TopRight, Style: bs})

	// Bottom border
	surface.SetCell(0, h-1, term.Cell{Ch: b.BottomLeft, Style: bs})
	for x := 1; x < w-1; x++ {
		surface.SetCell(x, h-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(w-1, h-1, term.Cell{Ch: b.BottomRight, Style: bs})

	// Side borders
	for y := 1; y < h-1; y++ {
		surface.SetCell(0, y, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(w-1, y, term.Cell{Ch: b.Vertical, Style: bs})
	}

	// Title
	s.drawTitle(surface, 1, 0, w-2, s.RightTitle, b, bs)

	// Content (absolute coords for SetRect, local for Sub)
	r := s.GetRect()
	if s.Right != nil {
		cw := w - 2
		ch := h - 2
		if cw > 0 && ch > 0 {
			s.Right.SetRect(Rect{X: r.X + 1, Y: r.Y + 1, W: cw, H: ch})
			sub := surface.Sub(Rect{X: 1, Y: 1, W: cw, H: ch})
			s.Right.Render(sub)
		}
	}
}

func (s *SplitPanelWidget) drawTitle(surface *RenderSurface, startX, y, maxW int, title string, b term.BorderSet, bs term.Style) {
	if title == "" || maxW < 3 {
		return
	}
	runes := []rune(title)
	avail := maxW - 2
	if len(runes) > avail {
		runes = runes[:avail]
	}

	x := startX
	surface.SetCell(x, y, term.Cell{Ch: b.Horizontal, Style: bs})
	x++
	for _, ch := range runes {
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: bs})
		x++
	}
	surface.SetCell(x, y, term.Cell{Ch: b.Horizontal, Style: bs})
}

func (s *SplitPanelWidget) HandleEvent(ev tcell.Event) EventResult {
	return EventIgnored
}

func (s *SplitPanelWidget) DividerScreenX() int {
	r := s.GetRect()
	return r.X + s.DividerPos + 1
}
