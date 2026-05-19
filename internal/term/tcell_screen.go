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
	t.scr.SetContent(x, y, c.Ch, nil, mapStyle(c.Style))
}

func mapStyle(s Style) tcell.Style {
	base := tcell.StyleDefault
	switch s {
	case StyleStatusBar:
		return base.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite)
	case StyleActiveTab:
		return base.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite).Bold(true)
	case StyleInactiveTab:
		return base.Background(tcell.ColorDarkGray).Foreground(tcell.ColorSilver)
	case StyleActivityBar:
		return base.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorSilver)
	case StyleActivityBarActive:
		return base.Background(tcell.ColorDarkSlateGray).Foreground(tcell.ColorWhite).Bold(true)
	case StyleSidebarHeader:
		return base.Foreground(tcell.ColorWhite).Bold(true)
	case StyleSidebarItem:
		return base.Foreground(tcell.ColorSilver)
	case StyleSidebarSelected:
		return base.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	case StylePaletteBorder:
		return base.Foreground(tcell.ColorDarkCyan)
	case StylePaletteInput:
		return base.Foreground(tcell.ColorWhite)
	case StylePaletteItem:
		return base.Foreground(tcell.ColorSilver)
	case StylePaletteSelected:
		return base.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	case StyleLineNumber:
		return base.Foreground(tcell.ColorDarkGray)
	default:
		return base
	}
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
