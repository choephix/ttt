package ui

import (
	"fmt"
	"macro/internal/core/diff"
	"macro/internal/term"

	"github.com/gdamore/tcell/v2"
)

type DiffViewWidget struct {
	BaseWidget
	FilePath string
	Lines    []diff.DiffLine
	TopLine  int
	viewH    int
}

func NewDiffViewWidget(filePath string, fd diff.FileDiff) *DiffViewWidget {
	return &DiffViewWidget{
		FilePath: filePath,
		Lines:    fd.AllLines(),
	}
}

func (d *DiffViewWidget) Focusable() bool { return true }

func (d *DiffViewWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	d.viewH = h

	gutterW := 4
	dividerX := (w - 1) / 2
	leftStart := gutterW
	leftW := dividerX - gutterW
	rightStart := dividerX + 1 + gutterW
	rightW := w - rightStart

	if leftW < 1 || rightW < 1 {
		return
	}

	for y := 0; y < h; y++ {
		idx := d.TopLine + y
		surface.SetCell(dividerX, y, term.Cell{Ch: '│', Style: term.StyleBorder})

		if idx >= len(d.Lines) {
			for x := 0; x < dividerX; x++ {
				surface.SetCell(x, y, term.Cell{Ch: ' '})
			}
			for x := dividerX + 1; x < w; x++ {
				surface.SetCell(x, y, term.Cell{Ch: ' '})
			}
			continue
		}

		dl := d.Lines[idx]

		leftStyle := kindToStyle(dl.Left.Kind)
		rightStyle := kindToStyle(dl.Right.Kind)

		d.renderGutter(surface, 0, y, gutterW, dl.Left, leftStyle)
		d.renderGutter(surface, dividerX+1, y, gutterW, dl.Right, rightStyle)

		d.renderSide(surface, leftStart, y, leftW, dl.Left.Text, leftStyle)
		d.renderSide(surface, rightStart, y, rightW, dl.Right.Text, rightStyle)
	}
}

func (d *DiffViewWidget) renderGutter(surface *RenderSurface, x, y, w int, sl diff.SideLine, style term.Style) {
	num := ""
	if sl.Num > 0 {
		num = fmt.Sprintf("%*d", w-1, sl.Num)
	}
	runes := []rune(num)
	for i := 0; i < w; i++ {
		ch := ' '
		if i < len(runes) {
			ch = runes[i]
		}
		surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: term.StyleLineNumber})
	}
}

func (d *DiffViewWidget) renderSide(surface *RenderSurface, x, y, w int, text string, style term.Style) {
	runes := []rune(text)
	for i := 0; i < w; i++ {
		ch := ' '
		if i < len(runes) {
			ch = runes[i]
		}
		surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: style})
	}
}

func kindToStyle(k diff.LineKind) term.Style {
	switch k {
	case diff.Added:
		return term.StyleDiffAdded
	case diff.Deleted:
		return term.StyleDiffDeleted
	default:
		return term.StyleDefault
	}
}

func (d *DiffViewWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		switch tev.Key() {
		case tcell.KeyUp:
			if d.TopLine > 0 {
				d.TopLine--
			}
			return EventConsumed
		case tcell.KeyDown:
			max := len(d.Lines) - d.viewH
			if max < 0 {
				max = 0
			}
			if d.TopLine < max {
				d.TopLine++
			}
			return EventConsumed
		case tcell.KeyPgUp:
			d.TopLine -= d.viewH
			if d.TopLine < 0 {
				d.TopLine = 0
			}
			return EventConsumed
		case tcell.KeyPgDn:
			max := len(d.Lines) - d.viewH
			if max < 0 {
				max = 0
			}
			d.TopLine += d.viewH
			if d.TopLine > max {
				d.TopLine = max
			}
			return EventConsumed
		case tcell.KeyHome:
			d.TopLine = 0
			return EventConsumed
		case tcell.KeyEnd:
			max := len(d.Lines) - d.viewH
			if max < 0 {
				max = 0
			}
			d.TopLine = max
			return EventConsumed
		}
	case *tcell.EventMouse:
		btn := tev.Buttons()
		if btn&tcell.WheelUp != 0 {
			d.TopLine -= 3
			if d.TopLine < 0 {
				d.TopLine = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			max := len(d.Lines) - d.viewH
			if max < 0 {
				max = 0
			}
			d.TopLine += 3
			if d.TopLine > max {
				d.TopLine = max
			}
			return EventConsumed
		}
	}
	return EventIgnored
}
