package term

import (
	"github.com/gdamore/tcell/v2"
)

const StyleCount = 18

type StyleMap [StyleCount]tcell.Style

func DefaultStyleMap() StyleMap {
	var m StyleMap
	base := tcell.StyleDefault
	m[StyleDefault] = base
	m[StyleStatusBar] = base.Background(tcell.ColorSilver).Foreground(tcell.ColorBlack)
	m[StyleActiveTab] = base.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite).Bold(true)
	m[StyleInactiveTab] = base.Foreground(tcell.ColorGray)
	m[StyleActivityBar] = base.Foreground(tcell.ColorGray)
	m[StyleActivityBarActive] = base.Foreground(tcell.ColorDarkCyan).Bold(true)
	m[StyleSidebarHeader] = base.Foreground(tcell.ColorDarkCyan).Bold(true)
	m[StyleSidebarItem] = base.Foreground(tcell.ColorSilver)
	m[StyleSidebarSelected] = base.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
	m[StylePaletteBorder] = base.Foreground(tcell.ColorDarkCyan)
	m[StylePaletteInput] = base.Foreground(tcell.ColorWhite)
	m[StylePaletteItem] = base.Foreground(tcell.ColorSilver)
	m[StylePaletteSelected] = base.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
	m[StyleLineNumber] = base.Foreground(tcell.ColorGray)
	m[StyleResizeHandle] = base.Foreground(tcell.ColorDarkCyan)
	m[StyleMenuBar] = base.Background(tcell.ColorSilver).Foreground(tcell.ColorBlack)
	m[StyleMenuBarActive] = base.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite).Bold(true)
	m[StyleBorder] = base.Foreground(tcell.ColorDarkCyan)
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
