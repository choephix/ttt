package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type DialogButton struct {
	Label   string
	Handler func()
}

type DialogWidget struct {
	BaseWidget
	Title     string
	Content   Widget
	Buttons   []DialogButton
	BoxWidth  int
	Borders   term.BorderSet
	OnDismiss func()

	focusBtn  bool // true when buttons have focus, false when content has focus
	selected  int
	boxX      int
	boxY      int
	boxW      int
	boxH      int
	btnHits   []hitRegion
	focus     *FocusManager
}

type hitRegion struct {
	X, Y, W int
}

func (h hitRegion) Contains(mx, my int) bool {
	return my == h.Y && mx >= h.X && mx < h.X+h.W
}

func NewDialogWidget(width int) *DialogWidget {
	d := &DialogWidget{
		BoxWidth: width,
		Borders:  term.DoubleBorderSet(),
		focus:    NewFocusManager(),
	}
	if d.BoxWidth <= 0 {
		d.BoxWidth = 40
	}
	return d
}

func (d *DialogWidget) SetContent(w Widget) {
	d.Content = w
	d.focus.Collect(w)
}

func (d *DialogWidget) Height() int { return 0 }
func (d *DialogWidget) Width() int  { return d.BoxWidth }

func (d *DialogWidget) CursorPosition() (int, int, bool) {
	if !d.focusBtn {
		if fw := d.focus.Focused(); fw != nil {
			if cp, ok := fw.(CursorPositioner); ok {
				return cp.CursorPosition()
			}
		}
	}
	return 0, 0, false
}

func (d *DialogWidget) contentHeight() int {
	if d.Content != nil {
		h := d.Content.Height()
		if h > 0 {
			return h
		}
	}
	return 1
}

func (d *DialogWidget) Render(surface Surface) {
	sw, sh := surface.Size()
	if sw <= 4 || sh <= 4 {
		return
	}

	d.boxW = d.BoxWidth
	if d.boxW > sw-4 {
		d.boxW = sw - 4
	}

	titleH := 0
	if d.Title != "" {
		titleH = 2
	}
	contentH := d.contentHeight()
	btnH := 0
	if len(d.Buttons) > 0 {
		btnH = 2
	}

	d.boxH = 2 + titleH + contentH + btnH
	d.boxX = (sw - d.boxW) / 2
	d.boxY = 2

	if d.boxY+d.boxH > sh {
		d.boxH = sh - d.boxY
	}

	surface.ClearRect(d.boxX, d.boxY, d.boxW, d.boxH, term.StylePaletteItem)
	surface.DrawBorder(d.boxX, d.boxY, d.boxW, d.boxH, d.Borders, term.StyleBorder)

	innerX := d.boxX + 1
	innerW := d.boxW - 2
	y := d.boxY + 1

	if d.Title != "" {
		surface.DrawText(innerX+1, y, d.Title, innerW-1, term.StylePaletteItem)
		y += 2
	}

	if d.Content != nil && contentH > 0 {
		r := d.GetRect()
		contentRect := Rect{X: r.X + innerX, Y: r.Y + y, W: innerW, H: contentH}
		d.Content.SetRect(contentRect)
		contentSurface := surface.Sub(Rect{X: innerX, Y: y - d.boxY, W: innerW, H: contentH})
		d.Content.Render(contentSurface)
		y += contentH
	}

	if len(d.Buttons) > 0 {
		y++
		d.renderButtons(surface, innerX, y, innerW)
	}
}

func (d *DialogWidget) renderButtons(surface Surface, innerX, btnY, innerW int) {
	labels := make([]string, len(d.Buttons))
	totalW := 0
	for i, btn := range d.Buttons {
		labels[i] = " " + btn.Label + " "
		totalW += len([]rune(labels[i]))
	}
	totalW += (len(labels) - 1) * 2

	startX := innerX + innerW - totalW - 1

	d.btnHits = make([]hitRegion, len(labels))
	bx := startX
	for i, label := range labels {
		style := term.StyleMuted
		if d.selected == i {
			style = term.StylePaletteSelected
		}
		runes := []rune(label)
		d.btnHits[i] = hitRegion{X: bx, Y: btnY, W: len(runes)}
		for j, ch := range runes {
			cell := term.Cell{Ch: ch, Style: style}
			if j == 1 {
				cell.Underline = true
			}
			surface.SetCell(bx+j, btnY, cell)
		}
		bx += len(runes) + 2
	}
}

func (d *DialogWidget) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		return d.handleMouse(e)
	case *tcell.EventKey:
		return d.handleKey(e)
	}
	return true
}

func (d *DialogWidget) handleMouse(ev *tcell.EventMouse) bool {
	if ev.Buttons()&tcell.Button1 != 0 {
		mx, my := ev.Position()
		for i, hit := range d.btnHits {
			if hit.Contains(mx, my) {
				d.selected = i
				if d.Buttons[i].Handler != nil {
					d.Buttons[i].Handler()
				}
				return true
			}
		}
	}
	if d.Content != nil {
		d.Content.HandleEvent(ev)
	}
	return true
}

func (d *DialogWidget) handleKey(ev *tcell.EventKey) bool {
	n := len(d.Buttons)

	switch ev.Key() {
	case tcell.KeyEscape:
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
		return true
	case tcell.KeyTab:
		if d.focusBtn && n > 0 {
			if d.selected < n-1 {
				d.selected++
				return true
			}
			d.focusBtn = false
			d.selected = 0
			return true
		}
		if !d.focusBtn && d.Content != nil && d.focus.HasNext() {
			d.focus.FocusNext()
			return true
		}
		if n > 0 {
			d.focusBtn = true
			d.selected = 0
		}
		return true
	case tcell.KeyBacktab:
		if d.focusBtn && n > 0 {
			if d.selected > 0 {
				d.selected--
				return true
			}
			d.focusBtn = false
			return true
		}
		if !d.focusBtn && d.Content != nil && d.focus.HasPrev() {
			d.focus.FocusPrev()
			return true
		}
		if n > 0 {
			d.focusBtn = true
			d.selected = n - 1
		}
		return true
	case tcell.KeyEnter:
		if d.focusBtn && n > 0 && d.selected >= 0 && d.selected < n && d.Buttons[d.selected].Handler != nil {
			d.Buttons[d.selected].Handler()
			return true
		}
	}

	if d.focusBtn {
		switch ev.Key() {
		case tcell.KeyLeft:
			if n > 0 {
				d.selected = (d.selected + n - 1) % n
			}
			return true
		case tcell.KeyRight:
			if n > 0 {
				d.selected = (d.selected + 1) % n
			}
			return true
		}
		return true
	}

	if d.Content != nil {
		if fw := d.focus.Focused(); fw != nil {
			fw.HandleEvent(ev)
			return true
		}
	}

	return true
}
