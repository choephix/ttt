package ui

import "github.com/eugenioenko/ttt/internal/term"

type RenderSurface struct {
	cells [][]term.Cell
	clip  Rect
}

func NewRenderSurface(cells [][]term.Cell, clip Rect) *RenderSurface {
	return &RenderSurface{cells: cells, clip: clip}
}

func (s *RenderSurface) Size() (w, h int) {
	return s.clip.W, s.clip.H
}

func (s *RenderSurface) SetCell(x, y int, c term.Cell) {
	absX := s.clip.X + x
	absY := s.clip.Y + y
	if absX < s.clip.X || absX >= s.clip.X+s.clip.W {
		return
	}
	if absY < s.clip.Y || absY >= s.clip.Y+s.clip.H {
		return
	}
	if absY < 0 || absY >= len(s.cells) {
		return
	}
	if absX < 0 || absX >= len(s.cells[absY]) {
		return
	}
	s.cells[absY][absX] = c
}

func (s *RenderSurface) Fill(c term.Cell) {
	for y := 0; y < s.clip.H; y++ {
		for x := 0; x < s.clip.W; x++ {
			s.SetCell(x, y, c)
		}
	}
}

func (s *RenderSurface) DrawText(x, y int, text string, maxW int, style term.Style) int {
	for _, ch := range text {
		if maxW > 0 && x >= maxW {
			break
		}
		s.SetCell(x, y, term.Cell{Ch: ch, Style: style})
		x++
	}
	return x
}

func (s *RenderSurface) ClearRect(x, y, w, h int, style term.Style) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.SetCell(x+dx, y+dy, term.Cell{Ch: ' ', Style: style})
		}
	}
}

func (s *RenderSurface) DrawBorder(x, y, w, h int, b term.BorderSet, style term.Style) {
	for bx := x; bx < x+w; bx++ {
		s.SetCell(bx, y, term.Cell{Ch: b.Horizontal, Style: style})
		s.SetCell(bx, y+h-1, term.Cell{Ch: b.Horizontal, Style: style})
	}
	for by := y; by < y+h; by++ {
		s.SetCell(x, by, term.Cell{Ch: b.Vertical, Style: style})
		s.SetCell(x+w-1, by, term.Cell{Ch: b.Vertical, Style: style})
	}
	s.SetCell(x, y, term.Cell{Ch: b.TopLeft, Style: style})
	s.SetCell(x+w-1, y, term.Cell{Ch: b.TopRight, Style: style})
	s.SetCell(x, y+h-1, term.Cell{Ch: b.BottomLeft, Style: style})
	s.SetCell(x+w-1, y+h-1, term.Cell{Ch: b.BottomRight, Style: style})
}

func (s *RenderSurface) Sub(r Rect) *RenderSurface {
	newX := s.clip.X + r.X
	newY := s.clip.Y + r.Y
	newW := r.W
	newH := r.H

	// Clamp to parent bounds
	if newX < s.clip.X {
		newW -= s.clip.X - newX
		newX = s.clip.X
	}
	if newY < s.clip.Y {
		newH -= s.clip.Y - newY
		newY = s.clip.Y
	}
	if newX+newW > s.clip.X+s.clip.W {
		newW = s.clip.X + s.clip.W - newX
	}
	if newY+newH > s.clip.Y+s.clip.H {
		newH = s.clip.Y + s.clip.H - newY
	}
	if newW < 0 {
		newW = 0
	}
	if newH < 0 {
		newH = 0
	}

	return &RenderSurface{
		cells: s.cells,
		clip:  Rect{X: newX, Y: newY, W: newW, H: newH},
	}
}
