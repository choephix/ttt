package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type DrawerConfig struct {
	Width     int
	MinWidth  int
	Borders   term.BorderSet
	OnDismiss func()
}

type DrawerWidget struct {
	BaseWidget
	Config  DrawerConfig
	Content Widget
	width   int
	dragging bool
	wasPressed bool
}

func NewDrawerWidget(config DrawerConfig) *DrawerWidget {
	if config.Width <= 0 {
		config.Width = 40
	}
	if config.MinWidth <= 0 {
		config.MinWidth = 20
	}
	return &DrawerWidget{
		Config: config,
		width:  config.Width,
	}
}

func (d *DrawerWidget) SetContent(w Widget) {
	d.Content = w
}

func (d *DrawerWidget) Reset() {
	d.width = d.Config.Width
}

func (d *DrawerWidget) Height() int { return 0 }
func (d *DrawerWidget) Width() int  { return 0 }

func (d *DrawerWidget) Render(surface Surface) {
	sw, sh := surface.Size()
	if sw <= 4 || sh <= 2 {
		return
	}

	w := d.width
	if w > sw-2 {
		w = sw - 2
	}
	if w < d.Config.MinWidth {
		w = d.Config.MinWidth
	}

	x := sw - w
	b := d.Config.Borders
	bs := term.StyleBorder

	surface.SetCell(x, 0, term.Cell{Ch: b.TopLeft, Style: bs})
	for ix := x + 1; ix < sw-1; ix++ {
		surface.SetCell(ix, 0, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(sw-1, 0, term.Cell{Ch: b.TopRight, Style: bs})

	surface.SetCell(x, sh-1, term.Cell{Ch: b.BottomLeft, Style: bs})
	for ix := x + 1; ix < sw-1; ix++ {
		surface.SetCell(ix, sh-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	surface.SetCell(sw-1, sh-1, term.Cell{Ch: b.BottomRight, Style: bs})

	for iy := 1; iy < sh-1; iy++ {
		surface.SetCell(x, iy, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(sw-1, iy, term.Cell{Ch: b.Vertical, Style: bs})
	}

	innerX := x + 1
	innerW := w - 2
	if innerW <= 0 {
		return
	}

	for ix := range innerW {
		for iy := 1; iy < sh-1; iy++ {
			surface.SetCell(innerX+ix, iy, term.Cell{Ch: ' ', Style: term.StyleDefault})
		}
	}

	contentH := sh - 2
	if d.Content != nil && innerW > 0 && contentH > 0 {
		d.Content.SetRect(Rect{X: innerX, Y: 1, W: innerW, H: contentH})
		contentSurface := surface.Sub(Rect{X: innerX, Y: 1, W: innerW, H: contentH})
		d.Content.Render(contentSurface)
	}
}

func (d *DrawerWidget) HandleEvent(ev tcell.Event) bool {
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		if kev, ok := ev.(*tcell.EventKey); ok {
			if kev.Key() == tcell.KeyEscape {
				if d.Config.OnDismiss != nil {
					d.Config.OnDismiss()
				}
				return true
			}
			if d.Content != nil {
				return d.Content.HandleEvent(ev)
			}
		}
		return false
	}

	r := d.GetRect()
	mx, _ := mev.Position()
	btn := mev.Buttons()
	pressed := btn&tcell.Button1 != 0
	freshClick := pressed && !d.wasPressed
	d.wasPressed = pressed

	sw := r.W
	w := d.width
	if w > sw-2 {
		w = sw - 2
	}
	borderX := r.X + sw - w

	if d.dragging {
		if pressed {
			newW := r.X + sw - mx
			if newW < d.Config.MinWidth {
				newW = d.Config.MinWidth
			}
			if newW > sw-2 {
				newW = sw - 2
			}
			d.width = newW
			return true
		}
		d.dragging = false
		if d.width < d.Config.MinWidth {
			d.width = d.Config.MinWidth
		}
		return true
	}

	if freshClick && mx >= borderX-1 && mx <= borderX {
		d.dragging = true
		return true
	}

	if freshClick && mx < borderX {
		if d.Config.OnDismiss != nil {
			d.Config.OnDismiss()
		}
		return true
	}

	if d.Content != nil {
		return d.Content.HandleEvent(ev)
	}
	return true
}
