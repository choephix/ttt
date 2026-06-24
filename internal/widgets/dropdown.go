package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type DropdownConfig struct {
	Icon    string      `json:"icon,omitempty"`
	Entries []MenuEntry `json:"entries,omitempty"`
	Padded  bool        `json:"padded,omitempty"`
	Style   term.Style  `json:"-"`

	OnMenu func(entries []MenuEntry, screenX, screenY int)
}

type DropdownWidget struct {
	Config DropdownConfig
	rect   Rect
}

func NewDropdownWidget(config DropdownConfig) *DropdownWidget {
	if config.Icon == "" {
		config.Icon = "⋮"
	}
	return &DropdownWidget{Config: config}
}

func (d *DropdownWidget) SetRect(r Rect) {
	d.rect = r
}

func (d *DropdownWidget) GetRect() Rect {
	return d.rect
}

func (d *DropdownWidget) Width() int {
	w := 0
	for range d.Config.Icon {
		w++
	}
	if d.Config.Padded {
		w += 2
	}
	return w
}

func (d *DropdownWidget) Render(surface Surface, x, y int, style term.Style) {
	if style == 0 {
		style = d.Config.Style
	}
	cx := x
	if d.Config.Padded {
		surface.SetCell(cx, y, term.Cell{Ch: ' ', Style: style})
		cx++
	}
	for _, ch := range d.Config.Icon {
		surface.SetCell(cx, y, term.Cell{Ch: ch, Style: style})
		cx++
	}
	if d.Config.Padded {
		surface.SetCell(cx, y, term.Cell{Ch: ' ', Style: style})
	}
}

func (d *DropdownWidget) HandleClick(x, y int) bool {
	if d.Config.OnMenu != nil && len(d.Config.Entries) > 0 {
		d.Config.OnMenu(d.Config.Entries, d.rect.X+x, d.rect.Y+y)
		return true
	}
	return false
}

func (d *DropdownWidget) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		if e.Buttons() == tcell.Button1 {
			mx, my := e.Position()
			if mx >= d.rect.X && mx < d.rect.X+d.rect.W && my >= d.rect.Y && my < d.rect.Y+d.rect.H {
				return d.HandleClick(mx-d.rect.X, my-d.rect.Y)
			}
		}
	}
	return false
}
