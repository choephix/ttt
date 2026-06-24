package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TitleConfig struct {
	Title  string      `json:"title"`
	Menu   []MenuEntry `json:"menu,omitempty"`
	Icon   string      `json:"icon,omitempty"`
	Padded bool        `json:"padded,omitempty"`
	Style  term.Style  `json:"-"`

	OnMenu func(entries []MenuEntry, screenX, screenY int)
}

type TitleWidget struct {
	BaseWidget
	Config   TitleConfig
	dropdown *DropdownWidget
}

func NewTitleWidget(config TitleConfig) *TitleWidget {
	t := &TitleWidget{Config: config}
	if len(config.Menu) > 0 {
		icon := config.Icon
		if icon == "" {
			icon = "⋮"
		}
		t.dropdown = NewDropdownWidget(DropdownConfig{
			Icon:    icon,
			Padded:  config.Padded,
			Entries: config.Menu,
			Style:   config.Style,
			OnMenu:  config.OnMenu,
		})
	}
	return t
}

func (t *TitleWidget) Height() int { return 1 + t.BoxOverheadH() }
func (t *TitleWidget) Width() int  { return 0 }

func (t *TitleWidget) Render(surface Surface) {
	inner := t.RenderBox(surface)
	w, _ := inner.Size()
	if w <= 0 {
		return
	}
	style := t.Config.Style

	maxTextW := w
	if t.dropdown != nil {
		dw := t.dropdown.Width()
		maxTextW = w - dw
		t.dropdown.Render(inner, w-dw, 0, style)
	}
	inner.DrawText(0, 0, t.Config.Title, maxTextW, style)
}

func (t *TitleWidget) HandleEvent(ev tcell.Event) bool {
	if t.dropdown != nil {
		switch e := ev.(type) {
		case *tcell.EventMouse:
			if e.Buttons() == tcell.Button1 {
				mx, my := e.Position()
				r := t.GetRect()
				if my == r.Y {
					dw := t.dropdown.Width()
					iconStart := r.X + r.W - dw
					if mx >= iconStart && mx < r.X+r.W {
						if t.Config.OnMenu != nil {
							t.Config.OnMenu(t.Config.Menu, mx, my)
						}
						return true
					}
				}
			}
		}
	}
	return false
}
