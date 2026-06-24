package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type Rect struct {
	X, Y, W, H int
}

type Widget interface {
	Height() int
	Width() int
	SetRect(r Rect)
	GetRect() Rect
	Render(surface Surface)
	HandleEvent(ev tcell.Event) bool
}

type Surface interface {
	Size() (w, h int)
	SetCell(x, y int, c term.Cell)
	DrawText(x, y int, text string, maxW int, style term.Style) int
	DrawBorder(x, y, w, h int, b term.BorderSet, style term.Style)
	ClearRect(x, y, w, h int, style term.Style)
	Fill(c term.Cell)
	Sub(r Rect) Surface
}
