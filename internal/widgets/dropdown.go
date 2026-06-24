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
	BaseWidget
	Config DropdownConfig
}

func NewDropdownWidget(config DropdownConfig) *DropdownWidget {
	if config.Icon == "" {
		config.Icon = "⋮"
	}
	return &DropdownWidget{Config: config}
}

func (d *DropdownWidget) Height() int { return 1 }

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
		r := d.GetRect()
		d.Config.OnMenu(d.Config.Entries, r.X+x, r.Y+y)
		return true
	}
	return false
}

func (d *DropdownWidget) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		if e.Buttons() == tcell.Button1 {
			mx, my := e.Position()
			r := d.GetRect()
			if mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H {
				return d.HandleClick(mx-r.X, my-r.Y)
			}
		}
	}
	return false
}
