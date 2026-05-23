package term

import (
	"github.com/gdamore/tcell/v2"
)

const StyleCount = 39

type StyleMap [StyleCount]tcell.Style

func DefaultStyleMap() StyleMap {
	var m StyleMap
	for i := range m {
		m[i] = tcell.StyleDefault
	}
	m[StyleSelection] = tcell.StyleDefault.Reverse(true)
	return m
}

// TcellScreen implements the Screen interface using tcell.
type TcellScreen struct {
	scr      tcell.Screen
	styleMap StyleMap
}

func NewTcellScreen() (*TcellScreen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	s.EnableMouse()
	return &TcellScreen{scr: s, styleMap: DefaultStyleMap()}, nil
}

func (t *TcellScreen) SetStyleMap(m StyleMap) {
	t.styleMap = m
}

func (t *TcellScreen) GetStyleMap() StyleMap {
	return t.styleMap
}

func (t *TcellScreen) Size() (w, h int) {
	return t.scr.Size()
}

func (t *TcellScreen) SetCell(x, y int, c Cell) {
	t.scr.SetContent(x, y, c.Ch, nil, t.styleMap[c.Style])
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

func (t *TcellScreen) HideCursor() {
	t.scr.HideCursor()
}
