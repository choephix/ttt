package term

import (
	"github.com/gdamore/tcell/v2"
)

// TcellScreen implements the Screen interface using tcell.
type TcellScreen struct {
	scr tcell.Screen
}

func NewTcellScreen() (*TcellScreen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	return &TcellScreen{scr: s}, nil
}

func (t *TcellScreen) Size() (w, h int) {
	return t.scr.Size()
}

func (t *TcellScreen) SetCell(x, y int, c Cell) {
	t.scr.SetContent(x, y, c.Ch, nil, tcell.StyleDefault)
}

func (t *TcellScreen) Show() {
	t.scr.Show()
}

func (t *TcellScreen) Clear() {
	t.scr.Clear()
}

func (t *TcellScreen) PollEvent() tcell.Event {
	return t.scr.PollEvent()
}

func (t *TcellScreen) Fini() {
	t.scr.Fini()
}

func (t *TcellScreen) ShowCursor(x, y int) {
	t.scr.ShowCursor(x, y)
}
