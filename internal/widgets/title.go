package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TitleConfig struct {
	Title  string      `json:"title"`
	Badge  string      `json:"badge,omitempty"`
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
		label := config.Icon
		if label == "" {
			label = "⋮"
		}
		var box *BoxModel
		if config.Padded {
			box = &BoxModel{PaddingLeft: 1, PaddingRight: 1}
		} else {
			box = &BoxModel{PaddingLeft: 0, PaddingRight: 0}
		}
		t.dropdown = NewDropdownWidget(DropdownConfig{
			Label:   label,
			Entries: config.Menu,
			Style:   config.Style,
			Box:     box,
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
		r := t.GetRect()
		ox := t.Box.MarginLeft + t.Box.PaddingLeft
		oy := t.Box.MarginTop + t.Box.PaddingTop
		if t.Box.BorderLeft {
			ox++
		}
		if t.Box.BorderTop {
			oy++
		}
		t.dropdown.SetRect(Rect{X: r.X + ox + w - dw, Y: r.Y + oy, W: dw, H: 1})
		ddSurface := inner.Sub(Rect{X: w - dw, Y: 0, W: dw, H: 1})
		t.dropdown.Render(ddSurface)
	}
	if t.Config.Badge != "" {
		badgeRunes := []rune(t.Config.Badge)
		bx := maxTextW - len(badgeRunes)
		if bx > 0 {
			inner.DrawText(bx, 0, t.Config.Badge, maxTextW, term.StyleMuted)
			maxTextW = bx - 1
		}
	}

	x := 0
	for _, ch := range t.Config.Title {
		if x >= maxTextW {
			break
		}
		inner.SetCell(x, 0, term.Cell{Ch: ch, Style: style, Bold: true})
		x++
	}
}

func (t *TitleWidget) HandleEvent(ev tcell.Event) EventResult {
	if t.dropdown != nil {
		if t.dropdown.HandleEvent(ev) == EventConsumed {
			return EventConsumed
		}
	}
	return EventIgnored
}
