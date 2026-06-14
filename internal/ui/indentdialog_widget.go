package ui

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type IndentDialogWidget struct {
	BaseWidget
	UseTabs   bool
	TabSize   int
	Borders   *term.BorderSet
	OnApply   func(useTabs bool, tabSize int)
	OnAuto    func()
	OnDismiss func()

	focusRow  int // 0=style, 1=sizes, 2=buttons
	focusCol  int
	spacesHit HitRegion
	tabsHit   HitRegion
	sizeHits  [6]HitRegion
	cancelHit HitRegion
	autoHit   HitRegion
	applyHit  HitRegion
}

var indentSizes = [6]int{1, 2, 3, 4, 6, 8}

func NewIndentDialogWidget(useTabs bool, tabSize int) *IndentDialogWidget {
	col := 0
	for i, s := range indentSizes {
		if s == tabSize {
			col = i
			break
		}
	}
	return &IndentDialogWidget{
		UseTabs:  useTabs,
		TabSize:  tabSize,
		focusRow: 1,
		focusCol: col,
	}
}

func (d *IndentDialogWidget) Focusable() bool { return true }

func (d *IndentDialogWidget) Render(surface *RenderSurface) {
	sw, _ := surface.Size()

	boxW := 34
	if boxW > sw-4 {
		boxW = sw - 4
	}
	boxH := 9
	boxX := (sw - boxW) / 2
	boxY := 2

	b := term.DoubleBorderSet()
	if d.Borders != nil {
		b = *d.Borders
	}
	surface.DrawBorder(boxX, boxY, boxW, boxH, b, term.StyleBorder)
	surface.ClearRect(boxX+1, boxY+1, boxW-2, boxH-2, term.StylePaletteItem)

	innerW := boxW - 2
	cx := boxX + 1

	// Row 0: Spaces / Tabs toggle
	styleY := boxY + 2
	spacesLabel := " Spaces "
	tabsLabel := " Tabs "
	toggleW := len(spacesLabel) + 2 + len(tabsLabel)
	toggleX := cx + (innerW-toggleW)/2

	spacesStyle := term.StyleInactiveTab
	tabsStyle := term.StyleInactiveTab
	if !d.UseTabs {
		spacesStyle = term.StyleActiveTab
	} else {
		tabsStyle = term.StyleActiveTab
	}

	d.spacesHit = HitRegion{X: toggleX, Y: styleY, W: len(spacesLabel)}
	for i, ch := range spacesLabel {
		surface.SetCell(toggleX+i, styleY, term.Cell{Ch: ch, Style: spacesStyle})
	}
	tabsX := toggleX + len(spacesLabel) + 2
	d.tabsHit = HitRegion{X: tabsX, Y: styleY, W: len(tabsLabel)}
	for i, ch := range tabsLabel {
		surface.SetCell(tabsX+i, styleY, term.Cell{Ch: ch, Style: tabsStyle})
	}

	// Row 1: Size buttons
	sizeY := boxY + 4
	sizeLabels := [6]string{" 1 ", " 2 ", " 3 ", " 4 ", " 6 ", " 8 "}
	totalSizeW := 0
	for _, l := range sizeLabels {
		totalSizeW += len(l)
	}
	totalSizeW += len(sizeLabels) - 1
	sizeStartX := cx + (innerW-totalSizeW)/2

	sx := sizeStartX
	for i, label := range sizeLabels {
		style := term.StyleInactiveTab
		if indentSizes[i] == d.TabSize {
			style = term.StyleActiveTab
		}
		d.sizeHits[i] = HitRegion{X: sx, Y: sizeY, W: len(label)}
		for j, ch := range label {
			surface.SetCell(sx+j, sizeY, term.Cell{Ch: ch, Style: style})
		}
		sx += len(label) + 1
	}

	// Row 2: Cancel / Auto / Apply buttons
	btnY := boxY + 6
	cancelLabel := " Cancel "
	autoLabel := " Auto "
	applyLabel := " Apply "
	btnTotalW := len(cancelLabel) + 2 + len(autoLabel) + 2 + len(applyLabel)
	btnStartX := cx + (innerW-btnTotalW)/2

	cancelStyle := term.StylePaletteItem
	autoStyle := term.StylePaletteItem
	applyStyle := term.StylePaletteItem
	if d.focusRow == 2 {
		switch d.focusCol {
		case 0:
			cancelStyle = term.StylePaletteSelected
		case 1:
			autoStyle = term.StylePaletteSelected
		case 2:
			applyStyle = term.StylePaletteSelected
		}
	}

	d.cancelHit = HitRegion{X: btnStartX, Y: btnY, W: len(cancelLabel)}
	for i, ch := range cancelLabel {
		surface.SetCell(btnStartX+i, btnY, term.Cell{Ch: ch, Style: cancelStyle})
	}

	autoX := btnStartX + len(cancelLabel) + 2
	d.autoHit = HitRegion{X: autoX, Y: btnY, W: len(autoLabel)}
	for i, ch := range autoLabel {
		surface.SetCell(autoX+i, btnY, term.Cell{Ch: ch, Style: autoStyle})
	}

	applyX := autoX + len(autoLabel) + 2
	d.applyHit = HitRegion{X: applyX, Y: btnY, W: len(applyLabel)}
	for i, ch := range applyLabel {
		surface.SetCell(applyX+i, btnY, term.Cell{Ch: ch, Style: applyStyle})
	}
}

func (d *IndentDialogWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, my := mev.Position()
			if d.spacesHit.Contains(mx, my) {
				d.UseTabs = false
				d.focusRow = 0
				d.focusCol = 0
				return EventConsumed
			}
			if d.tabsHit.Contains(mx, my) {
				d.UseTabs = true
				d.focusRow = 0
				d.focusCol = 1
				return EventConsumed
			}
			for i, hit := range d.sizeHits {
				if hit.Contains(mx, my) {
					d.TabSize = indentSizes[i]
					d.focusRow = 1
					d.focusCol = i
					return EventConsumed
				}
			}
			if d.cancelHit.Contains(mx, my) {
				if d.OnDismiss != nil {
					d.OnDismiss()
				}
				return EventConsumed
			}
			if d.autoHit.Contains(mx, my) {
				if d.OnAuto != nil {
					d.OnAuto()
				}
				return EventConsumed
			}
			if d.applyHit.Contains(mx, my) {
				if d.OnApply != nil {
					d.OnApply(d.UseTabs, d.TabSize)
				}
				return EventConsumed
			}
		}
		return EventConsumed
	}

	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventConsumed
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if d.OnDismiss != nil {
			d.OnDismiss()
		}
	case tcell.KeyUp:
		if d.focusRow > 0 {
			d.focusRow--
			d.clampFocusCol()
		}
	case tcell.KeyDown:
		if d.focusRow < 2 {
			d.focusRow++
			d.clampFocusCol()
		}
	case tcell.KeyLeft:
		if d.focusCol > 0 {
			d.focusCol--
		}
	case tcell.KeyRight:
		maxCol := d.maxColForRow()
		if d.focusCol < maxCol {
			d.focusCol++
		}
	case tcell.KeyTab:
		if d.focusRow < 2 {
			d.focusRow++
			d.clampFocusCol()
		} else {
			d.focusRow = 0
			d.clampFocusCol()
		}
	case tcell.KeyEnter:
		d.activateFocused()
	}

	return EventConsumed
}

func (d *IndentDialogWidget) maxColForRow() int {
	switch d.focusRow {
	case 0:
		return 1
	case 1:
		return 5
	case 2:
		return 2
	}
	return 0
}

func (d *IndentDialogWidget) clampFocusCol() {
	max := d.maxColForRow()
	if d.focusCol > max {
		d.focusCol = max
	}
}

func (d *IndentDialogWidget) activateFocused() {
	switch d.focusRow {
	case 0:
		d.UseTabs = d.focusCol == 1
	case 1:
		d.TabSize = indentSizes[d.focusCol]
	case 2:
		switch d.focusCol {
		case 0:
			if d.OnDismiss != nil {
				d.OnDismiss()
			}
		case 1:
			if d.OnAuto != nil {
				d.OnAuto()
			}
		case 2:
			if d.OnApply != nil {
				d.OnApply(d.UseTabs, d.TabSize)
			}
		}
	}
}
