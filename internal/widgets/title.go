package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TitleConfig struct {
	Title string      `json:"title"`
	Menu  []MenuEntry `json:"menu,omitempty"`
	Style term.Style  `json:"-"`

	OnMenu func(entries []MenuEntry, screenX, screenY int)
}

type TitleWidget struct {
	Config TitleConfig
	rect   Rect
}

func NewTitleWidget(config TitleConfig) *TitleWidget {
	return &TitleWidget{Config: config}
}

func (t *TitleWidget) SetRect(r Rect) {
	t.rect = r
}

func (t *TitleWidget) GetRect() Rect {
	return t.rect
}

func (t *TitleWidget) Render(surface Surface) {
	w, _ := surface.Size()
	style := t.Config.Style

	surface.DrawText(0, 0, t.Config.Title, w, style)

	if len(t.Config.Menu) > 0 {
		iconX := w - 1
		if iconX > 0 {
			surface.SetCell(iconX, 0, term.Cell{Ch: '⋮', Style: style})
		}
	}
}

func (t *TitleWidget) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		if e.Buttons() == tcell.Button1 && len(t.Config.Menu) > 0 {
			mx, my := e.Position()
			if my == t.rect.Y && mx == t.rect.X+t.rect.W-1 {
				if t.Config.OnMenu != nil {
					t.Config.OnMenu(t.Config.Menu, mx, my)
				}
				return true
			}
		}
	}
	return false
}
