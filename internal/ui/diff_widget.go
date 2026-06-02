package ui

import (
	"fmt"
	"sort"

	"github.com/eugenioenko/ttt/internal/core/diff"
	"github.com/eugenioenko/ttt/internal/core/highlight"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type diffMergedRef struct {
	isRight  bool
	sideIdx  int
}

type DiffViewWidget struct {
	BaseWidget
	FilePath    string
	Lines       []diff.DiffLine
	Highlighter *highlight.Highlighter
	TopLine     int
	LeftCol     int
	maxLineW    int
	viewH       int
	contentW    int
	scrollbar    Scrollbar
	hscrollbar   HScrollbar
	rhscrollbar  HScrollbar

	SearchMatchesLeft  []FindMatch
	SearchMatchesRight []FindMatch
	searchMergedRefs   []diffMergedRef
	searchActiveRight  bool
	searchActiveSideIdx int
}

func NewDiffViewWidget(filePath string, fd diff.FileDiff) *DiffViewWidget {
	lines := fd.AllLines()
	maxW := 0
	for _, dl := range lines {
		if lw := len([]rune(dl.Left.Text)); lw > maxW {
			maxW = lw
		}
		if rw := len([]rune(dl.Right.Text)); rw > maxW {
			maxW = rw
		}
	}
	return &DiffViewWidget{
		FilePath:            filePath,
		Lines:               lines,
		Highlighter:         highlight.New(filePath),
		maxLineW:            maxW,
		searchActiveSideIdx: -1,
	}
}

func (d *DiffViewWidget) Focusable() bool { return true }

func (d *DiffViewWidget) LeftLines() []string {
	lines := make([]string, len(d.Lines))
	for i, dl := range d.Lines {
		lines[i] = dl.Left.Text
	}
	return lines
}

func (d *DiffViewWidget) RightLines() []string {
	lines := make([]string, len(d.Lines))
	for i, dl := range d.Lines {
		lines[i] = dl.Right.Text
	}
	return lines
}

func (d *DiffViewWidget) CombinedLines() []string {
	lines := make([]string, len(d.Lines))
	for i, dl := range d.Lines {
		if dl.Left.Text == dl.Right.Text {
			lines[i] = dl.Left.Text
		} else {
			lines[i] = dl.Left.Text + " " + dl.Right.Text
		}
	}
	return lines
}

func (d *DiffViewWidget) ApplySearchHighlight(query string, opts SearchOptions) {
	if query == "" {
		return
	}
	leftMatches, _ := FindInLines(d.LeftLines(), query, opts)
	rightMatches, _ := FindInLines(d.RightLines(), query, opts)
	d.SetSearchMatches(leftMatches, rightMatches)
}

func (d *DiffViewWidget) ScrollToLine(line int) {
	if d.viewH <= 0 {
		d.TopLine = line
		return
	}
	if line < d.TopLine || line >= d.TopLine+d.viewH {
		d.TopLine = line - d.viewH/2
		if d.TopLine < 0 {
			d.TopLine = 0
		}
		max := len(d.Lines) - d.viewH
		if max < 0 {
			max = 0
		}
		if d.TopLine > max {
			d.TopLine = max
		}
	}
}

func (d *DiffViewWidget) ClearSearch() {
	d.SearchMatchesLeft = nil
	d.SearchMatchesRight = nil
	d.searchMergedRefs = nil
	d.searchActiveRight = false
	d.searchActiveSideIdx = -1
}

func (d *DiffViewWidget) SetSearchMatches(left, right []FindMatch) []FindMatch {
	d.SearchMatchesLeft = left
	d.SearchMatchesRight = right

	type entry struct {
		match   FindMatch
		isRight bool
		sideIdx int
	}
	var entries []entry
	for i, m := range left {
		entries = append(entries, entry{m, false, i})
	}
	for i, m := range right {
		entries = append(entries, entry{m, true, i})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].match.Line != entries[j].match.Line {
			return entries[i].match.Line < entries[j].match.Line
		}
		if entries[i].isRight != entries[j].isRight {
			return !entries[i].isRight
		}
		return entries[i].match.Col < entries[j].match.Col
	})

	merged := make([]FindMatch, len(entries))
	d.searchMergedRefs = make([]diffMergedRef, len(entries))
	for i, e := range entries {
		merged[i] = e.match
		d.searchMergedRefs[i] = diffMergedRef{isRight: e.isRight, sideIdx: e.sideIdx}
	}
	return merged
}

func (d *DiffViewWidget) SetActiveMatch(mergedIdx int) {
	if mergedIdx >= 0 && mergedIdx < len(d.searchMergedRefs) {
		ref := d.searchMergedRefs[mergedIdx]
		d.searchActiveRight = ref.isRight
		d.searchActiveSideIdx = ref.sideIdx
	} else {
		d.searchActiveRight = false
		d.searchActiveSideIdx = -1
	}
}

func (d *DiffViewWidget) gutterWidth() int {
	maxLine := 0
	for _, dl := range d.Lines {
		if dl.Left.Num > maxLine {
			maxLine = dl.Left.Num
		}
		if dl.Right.Num > maxLine {
			maxLine = dl.Right.Num
		}
	}
	digits := 1
	for n := maxLine; n >= 10; n /= 10 {
		digits++
	}
	return digits + 3
}

func (d *DiffViewWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	r := d.GetRect()

	gutterW := d.gutterWidth()

	showVScroll := len(d.Lines) > h
	contentW := (w - 1) / 2 - gutterW
	showHScroll := d.maxLineW > contentW

	if showHScroll {
		h--
	}
	if showVScroll {
		w--
	}

	d.viewH = h

	dividerX := (w - 1) / 2
	leftStart := gutterW
	leftW := dividerX - gutterW
	rightStart := dividerX + 1 + gutterW
	rightW := w - rightStart
	d.contentW = leftW

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

		var leftSpans, rightSpans []highlight.Span
		if d.Highlighter != nil {
			if dl.Left.Text != "" {
				leftSpans = d.Highlighter.HighlightLine(dl.Left.Text)
			}
			if dl.Right.Text != "" {
				rightSpans = d.Highlighter.HighlightLine(dl.Right.Text)
			}
		}
		leftActive := -1
		if !d.searchActiveRight {
			leftActive = d.searchActiveSideIdx
		}
		rightActive := -1
		if d.searchActiveRight {
			rightActive = d.searchActiveSideIdx
		}
		d.renderSide(surface, leftStart, y, leftW, dl.Left.Text, leftStyle, leftSpans, idx, d.SearchMatchesLeft, leftActive)
		d.renderSide(surface, rightStart, y, rightW, dl.Right.Text, rightStyle, rightSpans, idx, d.SearchMatchesRight, rightActive)
	}

	if showVScroll {
		d.scrollbar.X = r.X + w
		d.scrollbar.Y = r.Y
		d.scrollbar.Height = h
		d.scrollbar.TotalItems = len(d.Lines)
		d.scrollbar.TopItem = d.TopLine
		d.scrollbar.Render(surface, w, 0)
	}

	if showHScroll {
		d.hscrollbar.X = r.X + leftStart
		d.hscrollbar.Y = r.Y + h
		d.hscrollbar.Width = leftW
		d.hscrollbar.TotalCols = d.maxLineW
		d.hscrollbar.LeftCol = d.LeftCol
		d.hscrollbar.Render(surface, leftStart, h)

		for x := 0; x < gutterW; x++ {
			surface.SetCell(x, h, term.Cell{Ch: ' '})
		}
		surface.SetCell(dividerX, h, term.Cell{Ch: '│', Style: term.StyleBorder})
		for x := dividerX + 1; x < rightStart; x++ {
			surface.SetCell(x, h, term.Cell{Ch: ' '})
		}

		d.rhscrollbar.X = r.X + rightStart
		d.rhscrollbar.Y = r.Y + h
		d.rhscrollbar.Width = rightW
		d.rhscrollbar.TotalCols = d.maxLineW
		d.rhscrollbar.LeftCol = d.LeftCol
		d.rhscrollbar.Render(surface, rightStart, h)
	}
}

func (d *DiffViewWidget) renderGutter(surface *RenderSurface, x, y, w int, sl diff.SideLine, style term.Style) {
	num := ""
	if sl.Num > 0 {
		num = fmt.Sprintf("%d", sl.Num)
	}
	padded := " " + fmt.Sprintf("%*s", w-3, num) + "  "
	for i, ch := range []rune(padded) {
		if i >= w {
			break
		}
		surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: term.StyleLineNumber})
	}
}

func (d *DiffViewWidget) renderSide(surface *RenderSurface, x, y, w int, text string, baseStyle term.Style, spans []highlight.Span, lineIdx int, matches []FindMatch, activeIdx int) {
	runes := []rune(text)
	for i := 0; i < w; i++ {
		colIdx := d.LeftCol + i
		ch := ' '
		if colIdx < len(runes) {
			ch = runes[colIdx]
		}
		style := term.StyleDefault
		for _, sp := range spans {
			if colIdx >= sp.Start && colIdx < sp.End {
				style = sp.Style
				break
			}
		}
		for mi, m := range matches {
			if m.Line == lineIdx && colIdx >= m.Col && colIdx < m.Col+m.Len {
				if mi == activeIdx {
					style = term.StyleSearchActive
				} else {
					style = term.StyleSearchMatch
				}
				break
			}
		}
		cell := term.Cell{Ch: ch, Style: style}
		if style != term.StyleSearchMatch && style != term.StyleSearchActive && baseStyle != term.StyleDefault {
			cell.BgStyle = baseStyle
		}
		surface.SetCell(x+i, y, cell)
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

func (d *DiffViewWidget) clampLeftCol() {
	max := d.maxLineW - d.contentW
	if max < 0 {
		max = 0
	}
	if d.LeftCol > max {
		d.LeftCol = max
	}
	if d.LeftCol < 0 {
		d.LeftCol = 0
	}
}

func (d *DiffViewWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := d.scrollbar.HandleEvent(ev); consumed {
		d.TopLine = newTop
		return EventConsumed
	}
	if newLeft, consumed := d.hscrollbar.HandleEvent(ev); consumed {
		d.LeftCol = newLeft
		return EventConsumed
	}
	if newLeft, consumed := d.rhscrollbar.HandleEvent(ev); consumed {
		d.LeftCol = newLeft
		return EventConsumed
	}

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
		case tcell.KeyLeft:
			if d.LeftCol > 0 {
				d.LeftCol--
			}
			return EventConsumed
		case tcell.KeyRight:
			d.LeftCol++
			d.clampLeftCol()
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
			d.LeftCol = 0
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
		mod := tev.Modifiers()
		if btn&tcell.WheelUp != 0 {
			if mod&tcell.ModShift != 0 {
				d.LeftCol -= 4
				if d.LeftCol < 0 {
					d.LeftCol = 0
				}
			} else {
				d.TopLine -= 3
				if d.TopLine < 0 {
					d.TopLine = 0
				}
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			if mod&tcell.ModShift != 0 {
				d.LeftCol += 4
				d.clampLeftCol()
			} else {
				max := len(d.Lines) - d.viewH
				if max < 0 {
					max = 0
				}
				d.TopLine += 3
				if d.TopLine > max {
					d.TopLine = max
				}
			}
			return EventConsumed
		}
		if btn&tcell.WheelLeft != 0 {
			d.LeftCol -= 4
			if d.LeftCol < 0 {
				d.LeftCol = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelRight != 0 {
			d.LeftCol += 4
			d.clampLeftCol()
			return EventConsumed
		}
	}
	return EventIgnored
}
