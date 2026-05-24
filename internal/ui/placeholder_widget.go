package ui

import "github.com/eugenioenko/ttt/internal/term"

type PlaceholderWidget struct {
	BaseWidget
	Text string
}

func NewPlaceholderWidget(text string) *PlaceholderWidget {
	return &PlaceholderWidget{Text: text}
}

func (p *PlaceholderWidget) Focusable() bool { return true }

func (p *PlaceholderWidget) Render(surface *RenderSurface) {
	w, _ := surface.Size()
	for i, ch := range p.Text {
		if i >= w {
			break
		}
		surface.SetCell(i, 0, term.Cell{Ch: ch, Style: term.StyleDefault})
	}
}
