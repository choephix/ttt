package ui

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/core/cursor"
	"github.com/eugenioenko/ttt/internal/core/diff"
	"github.com/eugenioenko/ttt/internal/core/fold"
	"github.com/eugenioenko/ttt/internal/core/highlight"
	"github.com/eugenioenko/ttt/internal/core/multicursor"
	"github.com/eugenioenko/ttt/internal/core/selection"
	"github.com/eugenioenko/ttt/internal/core/undo"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/view"

	"github.com/gdamore/tcell/v2"
)

const maxBracketColorLines = 10_000

type EditorPaneWidget struct {
	BaseWidget
	Buf                     *buffer.Buffer
	Cursor                  *cursor.Cursor
	Viewport                *view.Viewport
	Undo                    *undo.UndoStack
	Selection               *selection.Selection
	CursorX                 int
	CursorY                 int
	TabSize                 int
	UseTabs                 bool
	LineNumbers             bool
	GutterStyle             string
	WordWrap                bool
	AutoIndent              bool
	BracketPairColorization bool
	BracketColorStyles      []term.Style
	Highlighter             *highlight.Highlighter
	SearchQuery             string
	SearchMatches           []FindMatch
	SearchActive            int
	lastClickTime           int64
	lastClickLine           int
	lastClickCol            int
	clickCount              int
	mouseDown               bool
	scrollbar               Scrollbar
	hscrollbar              HScrollbar
	Diagnostics             []Diagnostic
	Folds                   *fold.State
	OnChange                func()
	bufferDirty             bool
	Multi                   *multicursor.MultiCursor
	multiSearchWord         string
	gutterHover             bool
	gutterHoverLine         int
	mouseDownX, mouseDownY  int
	maxWidthSeen            int
	cachedVisibleLines      []int
	searchByLine            map[int][]int
	diagByLine              map[int][]int
	LineChanges             []diff.LineChangeKind
	bracketColorCache       bracketColorMap
	bracketColorDirty       bool
	wrapMap                 []wrapEntry
	wrapTopOffset           int
}

func NewEditorPaneWidget(buf *buffer.Buffer, cur *cursor.Cursor, vp *view.Viewport) *EditorPaneWidget {
	return &EditorPaneWidget{
		Buf:               buf,
		Cursor:            cur,
		Viewport:          vp,
		AutoIndent:        true,
		bracketColorDirty: true,
	}
}

func (e *EditorPaneWidget) InvalidateBracketColors() {
	e.bracketColorDirty = true
}

func (e *EditorPaneWidget) Focusable() bool { return true }

func (e *EditorPaneWidget) GutterWidth() int {
	if !e.LineNumbers {
		if e.GutterStyle != "minimal" {
			return 1
		}
		return 0
	}
	digits := len(strconv.Itoa(len(e.Buf.Lines)))
	if digits < 2 {
		digits = 2
	}
	switch e.GutterStyle {
	case "minimal":
		return digits + 1
	case "extended":
		return digits + 5
	default:
		return digits + 3
	}
}

func (e *EditorPaneWidget) computeMaxLineWidth() int {
	tabW := e.resolveTabSize()
	maxW := 0
	topLine := e.Viewport.TopLine
	botLine := topLine + e.Viewport.Height
	if botLine > len(e.Buf.Lines) {
		botLine = len(e.Buf.Lines)
	}
	for i := topLine; i < botLine; i++ {
		lw := bufColToVisualCol(e.Buf.Lines[i], len([]rune(e.Buf.Lines[i])), tabW)
		if lw > maxW {
			maxW = lw
		}
	}
	if maxW > e.maxWidthSeen {
		e.maxWidthSeen = maxW
	}
	return e.maxWidthSeen
}

func (e *EditorPaneWidget) clampLeftCol() {
	editorW := e.Viewport.Width
	maxW := e.computeMaxLineWidth()
	max := maxW - editorW
	if max < 0 {
		max = 0
	}
	if e.Viewport.LeftCol > max {
		e.Viewport.LeftCol = max
	}
	if e.Viewport.LeftCol < 0 {
		e.Viewport.LeftCol = 0
	}
}

func (e *EditorPaneWidget) buildSearchIndex() {
	e.searchByLine = make(map[int][]int, len(e.SearchMatches))
	for i, m := range e.SearchMatches {
		e.searchByLine[m.Line] = append(e.searchByLine[m.Line], i)
	}
}

func (e *EditorPaneWidget) buildDiagIndex() {
	e.diagByLine = make(map[int][]int)
	for i, d := range e.Diagnostics {
		for line := d.StartLine; line <= d.EndLine; line++ {
			e.diagByLine[line] = append(e.diagByLine[line], i)
		}
	}
}

func (e *EditorPaneWidget) hasFolds() bool {
	return !e.WordWrap && e.Folds != nil && e.Folds.HasCollapsedFolds()
}

func (e *EditorPaneWidget) ensureTopLineVisible() {
	if e.Folds == nil {
		return
	}
	if r := e.Folds.ContainingFold(e.Viewport.TopLine); r != nil {
		e.Viewport.TopLine = r.StartLine
	}
}

func (e *EditorPaneWidget) screenToBufferLine(y int) int {
	if e.cachedVisibleLines == nil || e.Folds == nil {
		return e.Viewport.TopLine + y
	}
	topVis := e.Folds.BufferToVisible(e.Viewport.TopLine)
	if topVis < 0 {
		topVis = 0
	}
	idx := topVis + y
	if idx < 0 {
		return 0
	}
	if idx >= len(e.cachedVisibleLines) {
		return len(e.Buf.Lines)
	}
	return e.cachedVisibleLines[idx]
}

func (e *EditorPaneWidget) Render(surface Surface) {
	w, h := surface.Size()

	totalLines := len(e.Buf.Lines)
	gutterW := e.GutterWidth()

	maxLineW := e.computeMaxLineWidth()
	tabW := e.resolveTabSize()

	editorW := w - gutterW
	showHScrollbar := !e.WordWrap && maxLineW > editorW
	if showHScrollbar {
		h--
	}

	foldsActive := e.hasFolds()
	if foldsActive {
		e.ensureTopLineVisible()
		e.cachedVisibleLines = e.Folds.VisibleLines(totalLines)
	} else {
		e.cachedVisibleLines = nil
	}

	visibleCount := totalLines
	if foldsActive {
		visibleCount = len(e.cachedVisibleLines)
	}

	if e.WordWrap {
		visibleCount = totalVisualLines(e.Buf.Lines, editorW, tabW)
	}

	showScrollbar := visibleCount > h
	if showScrollbar {
		editorW--
	}
	if editorW < 1 {
		editorW = 1
	}
	if e.WordWrap && editorW > 4 {
		switch e.GutterStyle {
		case "minimal":
			editorW--
		case "extended":
			editorW -= 3
		default:
			editorW -= 2
		}
	}

	e.Viewport.Width = editorW
	e.Viewport.Height = h

	if e.WordWrap {
		e.Viewport.LeftCol = 0
		visibleCount = totalVisualLines(e.Buf.Lines, editorW, tabW)
		showScrollbar = visibleCount > h
	}

	sel := e.Selection
	hasSel := sel != nil && sel.Active

	multiActive := e.isMultiActive()
	var allCursors []multicursor.CursorState
	if multiActive {
		e.syncToMulti()
		allCursors = e.Multi.Cursors
	}

	hasSearch := len(e.SearchMatches) > 0

	matchLine, matchCol, hasMatch := e.findMatchingBracket()

	if e.Viewport.TopLine < 0 {
		e.Viewport.TopLine = 0
	}

	var bracketColors bracketColorMap
	if e.BracketPairColorization && len(e.Buf.Lines) <= maxBracketColorLines {
		if e.bracketColorDirty {
			e.bracketColorCache = e.computeBracketColors()
			e.bracketColorDirty = false
		}
		bracketColors = e.bracketColorCache
	}

	if e.WordWrap {
		e.wrapMap = buildWrapMap(e.Buf.Lines, e.Viewport.TopLine, e.wrapTopOffset, h, editorW, tabW)
	} else {
		e.wrapMap = nil
	}

	for y := 0; y < h; y++ {
		var lineIdx int
		var segStartCol int
		var isWrapContinuation bool

		if e.WordWrap && e.wrapMap != nil {
			entry := e.wrapMap[y]
			lineIdx = entry.bufLine
			segStartCol = entry.startCol
			isWrapContinuation = segStartCol > 0
		} else {
			lineIdx = e.screenToBufferLine(y)
			segStartCol = 0
		}

		if gutterW > 0 {
			gutterStyle := term.StyleLineNumber
			if lineIdx < totalLines && lineIdx == e.Cursor.Line {
				gutterStyle = term.StyleActiveLine
			}
			var padded string
			if !e.LineNumbers || (e.WordWrap && isWrapContinuation) {
				padded = strings.Repeat(" ", gutterW)
			} else {
				numStr := ""
				if lineIdx < totalLines {
					numStr = strconv.Itoa(lineIdx + 1)
				}
				switch e.GutterStyle {
				case "minimal":
					padded = strings.Repeat(" ", gutterW-1-len(numStr)) + numStr + " "
				case "extended":
					padded = "  " + strings.Repeat(" ", gutterW-5-len(numStr)) + numStr + "   "
				default:
					padded = " " + strings.Repeat(" ", gutterW-3-len(numStr)) + numStr + "  "
				}
			}
			for i, ch := range padded {
				surface.SetCell(i, y, term.Cell{Ch: ch, Style: gutterStyle})
			}
			if e.Folds != nil && !e.WordWrap && lineIdx < totalLines && !isWrapContinuation {
				if fr := e.Folds.FoldAt(lineIdx); fr != nil {
					chevronCol := gutterW - 2
					collapsedCh := '▶'
					expandedCh := '▼'
					if e.GutterStyle == "minimal" {
						chevronCol = gutterW - 1
						collapsedCh = '▸'
						expandedCh = '▾'
					}
					if e.Folds.IsCollapsed(lineIdx) {
						surface.SetCell(chevronCol, y, term.Cell{Ch: collapsedCh, Style: gutterStyle})
					} else if e.gutterHover {
						surface.SetCell(chevronCol, y, term.Cell{Ch: expandedCh, Style: gutterStyle})
					}
				}
			}
			if lineIdx < totalLines && lineIdx < len(e.LineChanges) && !isWrapContinuation {
				change := e.LineChanges[lineIdx]
				if change != diff.LineUnchanged {
					var ch rune
					var style term.Style
					switch change {
					case diff.LineAdded:
						ch = '▎'
						style = term.StyleGutterAdded
					case diff.LineModified:
						ch = '▎'
						style = term.StyleGutterModified
					case diff.LineDeleted:
						ch = '▾'
						style = term.StyleGutterDeleted
					}
					surface.SetCell(0, y, term.Cell{Ch: ch, Style: style})
				}
			}
		}

		if lineIdx < totalLines {
			line := []rune(e.Buf.Lines[lineIdx])
			var syntaxSpans []highlight.Span
			if e.Highlighter != nil {
				syntaxSpans = e.Highlighter.HighlightLine(e.Buf.Lines[lineIdx])
			}

			isCollapsedLine := e.Folds != nil && e.Folds.IsCollapsed(lineIdx)
			var annRunes []rune
			if isCollapsedLine && !isWrapContinuation {
				annRunes = []rune(" ⋯")
			}

			var leftCol int
			if e.WordWrap {
				leftCol = bufColToVisualCol(e.Buf.Lines[lineIdx], segStartCol, tabW)
			} else {
				leftCol = e.Viewport.LeftCol
			}
			screenCells := e.renderLineToScreen(line, syntaxSpans, isCollapsedLine, annRunes, tabW, leftCol, editorW)
			var lineBrackets []bracketColorEntry
			if bracketColors != nil && lineIdx < len(bracketColors) {
				lineBrackets = bracketColors[lineIdx]
			}
			for x := 0; x < editorW; x++ {
				colIdx := screenCells[x].bufCol
				ch := screenCells[x].ch
				style := screenCells[x].style

				for _, bc := range lineBrackets {
					if bc.col == colIdx {
						style = bc.style
						break
					}
				}

				if hasSearch {
					if indices, ok := e.searchByLine[lineIdx]; ok {
						for _, mi := range indices {
							m := e.SearchMatches[mi]
							if colIdx >= m.Col && colIdx < m.Col+m.Len {
								if mi == e.SearchActive {
									style = term.StyleSearchActive
								} else {
									style = term.StyleSearchMatch
								}
								break
							}
						}
					}
				}
				isSearchHighlight := style == term.StyleSearchActive || style == term.StyleSearchMatch
				bgStyle := term.Style(0)
				inAnySel := false
				if multiActive {
					for _, mc := range allCursors {
						if mc.Sel.Active && mc.Sel.Contains(lineIdx, colIdx, mc.Line, mc.Col) {
							bgStyle = term.StyleSelection
							inAnySel = true
							break
						}
					}
				} else if hasSel && sel.Contains(lineIdx, colIdx, e.Cursor.Line, e.Cursor.Col) {
					bgStyle = term.StyleSelection
					inAnySel = true
				}
				if !inAnySel {
					isCursorLine := false
					if multiActive {
						for _, mc := range allCursors {
							if mc.Line == lineIdx {
								isCursorLine = true
								break
							}
						}
					} else {
						isCursorLine = lineIdx == e.Cursor.Line && !hasSel
					}
					if isCursorLine && !isSearchHighlight {
						bgStyle = term.StyleActiveLine
					}
				}
				if hasMatch && ((lineIdx == e.Cursor.Line && colIdx == e.Cursor.Col) ||
					(lineIdx == matchLine && colIdx == matchCol)) {
					bgStyle = term.StyleBracketMatch
				}
				if multiActive {
					for _, mc := range allCursors {
						if mc.Line == lineIdx && mc.Col == colIdx {
							style = term.StyleSelection
							bgStyle = 0
							break
						}
					}
				}
				ulStyle := e.diagStyleAt(lineIdx, colIdx)
				surface.SetCell(gutterW+x, y, term.Cell{Ch: ch, Style: style, BgStyle: bgStyle, UlStyle: ulStyle})
			}
		} else {
			for x := 0; x < editorW; x++ {
				surface.SetCell(gutterW+x, y, term.Cell{Ch: ' '})
			}
		}
	}

	scrollbarCol := w - 1
	if showHScrollbar {
		scrollbarCol = w - 1
	}
	if showScrollbar {
		r := e.GetRect()
		e.scrollbar.X = r.X + scrollbarCol
		e.scrollbar.Y = r.Y
		e.scrollbar.Height = h
		if e.WordWrap {
			e.scrollbar.TotalItems = visibleCount + h - 1
			curTopVisRow, _ := bufferPosToWrapScreenPos(e.Buf.Lines, e.Viewport.TopLine, 0, editorW, tabW)
			curTopVisRow += e.wrapTopOffset
			e.scrollbar.TopItem = curTopVisRow
		} else if foldsActive {
			e.scrollbar.TotalItems = visibleCount + h - 1
			topVis := e.Folds.BufferToVisible(e.Viewport.TopLine)
			if topVis < 0 {
				topVis = 0
			}
			e.scrollbar.TopItem = topVis
		} else {
			e.scrollbar.TotalItems = totalLines + h - 1
			e.scrollbar.TopItem = e.Viewport.TopLine
		}
		e.scrollbar.Render(surface, scrollbarCol, 0)
	}

	if showHScrollbar {
		r := e.GetRect()
		trackW := editorW
		e.hscrollbar.X = r.X + gutterW
		e.hscrollbar.Y = r.Y + h
		e.hscrollbar.Width = trackW
		e.hscrollbar.TotalCols = maxLineW
		e.hscrollbar.LeftCol = e.Viewport.LeftCol
		e.hscrollbar.Render(surface, gutterW, h)
	}

	r := e.GetRect()
	if e.WordWrap {
		curVisRow, curScreenCol := bufferPosToWrapScreenPos(e.Buf.Lines, e.Cursor.Line, e.Cursor.Col, editorW, tabW)
		topVisRow, _ := bufferPosToWrapScreenPos(e.Buf.Lines, e.Viewport.TopLine, 0, editorW, tabW)
		topVisRow += e.wrapTopOffset
		e.CursorX = curScreenCol + gutterW + r.X
		e.CursorY = curVisRow - topVisRow + r.Y
	} else {
		e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
		cursorVisCol := bufColToVisualCol(e.Buf.Lines[e.Cursor.Line], e.Cursor.Col, tabW)
		e.CursorX = cursorVisCol - e.Viewport.LeftCol + gutterW + r.X
		if foldsActive {
			curVis := e.Folds.BufferToVisible(e.Cursor.Line)
			topVis := e.Folds.BufferToVisible(e.Viewport.TopLine)
			if curVis >= 0 && topVis >= 0 {
				e.CursorY = curVis - topVis + r.Y
			} else {
				e.CursorY = e.Cursor.Line - e.Viewport.TopLine + r.Y
			}
		} else {
			e.CursorY = e.Cursor.Line - e.Viewport.TopLine + r.Y
		}
	}
}

func (e *EditorPaneWidget) DiagnosticAt(line, col int) *Diagnostic {
	for i := range e.Diagnostics {
		d := &e.Diagnostics[i]
		if line < d.StartLine || line > d.EndLine {
			continue
		}
		if line == d.StartLine && col < d.StartCol {
			continue
		}
		if line == d.EndLine && col >= d.EndCol {
			continue
		}
		return d
	}
	return nil
}

func (e *EditorPaneWidget) diagStyleAt(line, col int) term.Style {
	indices, ok := e.diagByLine[line]
	if !ok {
		return 0
	}
	for _, i := range indices {
		d := e.Diagnostics[i]
		if line == d.StartLine && col < d.StartCol {
			continue
		}
		if line == d.EndLine && col >= d.EndCol {
			continue
		}
		if d.Style != 0 {
			return d.Style
		}
		switch d.Severity {
		case DiagError:
			return term.StyleDiagError
		case DiagWarning:
			return term.StyleDiagWarning
		case DiagInformation:
			return term.StyleDiagInfo
		case DiagHint:
			return term.StyleDiagHint
		default:
			return term.StyleDiagError
		}
	}
	return 0
}

func (e *EditorPaneWidget) exec(cmd undo.EditCommand) {
	prevLines := len(e.Buf.Lines)
	cmd.Apply(e.Buf)
	if e.Undo != nil {
		e.Undo.Push(cmd)
	}
	e.bufferDirty = true
	if e.Folds != nil && len(e.Buf.Lines) != prevLines {
		e.Folds.SetRanges(fold.ComputeIndentRanges(e.Buf.Lines))
	}
}

func (e *EditorPaneWidget) ExecCommand(cmd undo.EditCommand) { e.exec(cmd) }

func (e *EditorPaneWidget) FlushOnChange() {
	if e.bufferDirty {
		e.bufferDirty = false
		e.maxWidthSeen = 0
		e.bracketColorDirty = true
		if e.Highlighter != nil {
			e.Highlighter.ClearCache()
		}
		if e.OnChange != nil {
			e.OnChange()
		}
	}
}

func (e *EditorPaneWidget) deleteSelection() {
	if e.Selection == nil || !e.Selection.Active {
		return
	}
	if e.Undo != nil {
		e.Undo.BreakGroup()
	}
	start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
	cmd := &undo.DeleteSelectionCommand{
		StartLine: start.Line, StartCol: start.Col,
		EndLine: end.Line, EndCol: end.Col,
	}
	e.exec(cmd)
	e.Cursor.Line = start.Line
	e.Cursor.Col = start.Col
	e.Selection.Clear()
}

func (e *EditorPaneWidget) replaceSelection(cmds ...undo.EditCommand) {
	if e.Selection == nil || !e.Selection.Active {
		return
	}
	start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
	all := make([]undo.EditCommand, 0, 1+len(cmds))
	all = append(all, &undo.DeleteSelectionCommand{
		StartLine: start.Line, StartCol: start.Col,
		EndLine: end.Line, EndCol: end.Col,
	})
	all = append(all, cmds...)
	if e.Undo != nil {
		e.Undo.BreakGroup()
	}
	e.exec(&undo.BatchCommand{Commands: all})
	if e.Undo != nil {
		e.Undo.ContinueGroup()
	}
	e.Cursor.Line = start.Line
	e.Cursor.Col = start.Col
	e.Selection.Clear()
}

func (e *EditorPaneWidget) startOrExtendSelection(shift bool) {
	if e.Selection == nil {
		return
	}
	if shift {
		if !e.Selection.Active {
			e.Selection.Start(e.Cursor.Line, e.Cursor.Col)
		}
	} else {
		e.Selection.Clear()
	}
}

func (e *EditorPaneWidget) pasteText(text string) {
	if e.Selection != nil && e.Selection.Active {
		e.deleteSelection()
	}
	e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Text: lines[0]})
		e.Cursor.Col += len([]rune(lines[0]))
	} else {
		currentLine := []rune(e.Buf.Lines[e.Cursor.Line])
		col := e.Cursor.Col
		if col > len(currentLine) {
			col = len(currentLine)
		}
		suffix := string(currentLine[col:])
		e.exec(&undo.PasteCommand{
			Line:   e.Cursor.Line,
			Col:    col,
			Text:   text,
			Suffix: suffix,
		})
		e.Cursor.Line += len(lines) - 1
		e.Cursor.Col = len([]rune(lines[len(lines)-1]))
	}
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) HandleEvent(ev tcell.Event) EventResult {
	if mev, ok := ev.(*tcell.EventMouse); ok {
		btn := mev.Buttons()

		if newTop, consumed := e.scrollbar.HandleEvent(ev); consumed {
			if e.Folds != nil && e.Folds.HasCollapsedFolds() {
				e.Viewport.TopLine = e.Folds.VisibleToBuffer(newTop)
			} else {
				e.Viewport.TopLine = newTop
			}
			if e.scrollbar.IsDragging() {
				return EventCaptured
			}
			return EventConsumed
		}
		if e.scrollbar.IsDragging() {
			return EventCaptured
		}
		if newLeft, consumed := e.hscrollbar.HandleEvent(ev); consumed {
			e.Viewport.LeftCol = newLeft
			e.clampLeftCol()
			if e.hscrollbar.IsDragging() {
				return EventCaptured
			}
			return EventConsumed
		}
		if e.hscrollbar.IsDragging() {
			return EventCaptured
		}

		mod := mev.Modifiers()
		if btn&tcell.WheelUp != 0 {
			if mod&tcell.ModShift != 0 {
				e.Viewport.LeftCol -= 4
				e.clampLeftCol()
			} else {
				e.scrollUp(3)
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			if mod&tcell.ModShift != 0 {
				e.Viewport.LeftCol += 4
				e.clampLeftCol()
			} else {
				e.scrollDown(3)
			}
			return EventConsumed
		}
		if btn&tcell.WheelLeft != 0 {
			e.Viewport.LeftCol -= 4
			e.clampLeftCol()
			return EventConsumed
		}
		if btn&tcell.WheelRight != 0 {
			e.Viewport.LeftCol += 4
			e.clampLeftCol()
			return EventConsumed
		}

		r := e.GetRect()
		mx, my := mev.Position()
		gutterW := e.GutterWidth()
		inGutter := gutterW > 0 && mx >= r.X && mx < r.X+gutterW

		if btn == tcell.ButtonNone && !e.mouseDown {
			prevHover := e.gutterHover
			prevLine := e.gutterHoverLine
			if inGutter {
				screenY := my - r.Y
				bufLine := e.screenToBufferLine(screenY)
				e.gutterHover = true
				e.gutterHoverLine = bufLine
			} else {
				e.gutterHover = false
			}
			if e.gutterHover != prevHover || e.gutterHoverLine != prevLine {
				return EventConsumed
			}
		}

		if btn&tcell.Button1 != 0 {
			if e.Undo != nil {
				e.Undo.BreakGroup()
			}
			line, col := e.mouseToPos(r, mx, my)

			isAlt := mev.Modifiers()&tcell.ModAlt != 0
			if isAlt && !e.mouseDown {
				e.ensureMulti()
				e.syncToMulti()
				e.Multi.Add(line, col)
				e.syncFromMulti()
				e.scrollViewport()
				return EventCaptured
			}

			if !e.mouseDown {
				e.mouseDown = true
				e.mouseDownX = mx
				e.mouseDownY = my

				if e.isMultiActive() {
					e.collapseMulti()
				}

				now := time.Now().UnixMilli()
				if now-e.lastClickTime < DoubleClickMs && line == e.lastClickLine && col == e.lastClickCol {
					e.clickCount++
				} else {
					e.clickCount = 1
				}
				e.lastClickTime = now
				e.lastClickLine = line
				e.lastClickCol = col

				switch e.clickCount {
				case 2:
					e.selectWord(line, col)
				case 3:
					e.selectLine(line)
					e.clickCount = 0
				default:
					if e.Selection != nil {
						e.Selection.Clear()
						e.Selection.Start(line, col)
					}
					e.Cursor.Line = line
					e.Cursor.Col = col
				}
			} else {
				e.Cursor.Line = line
				e.Cursor.Col = col
			}
			e.scrollViewport()
			return EventCaptured
		}
		if btn == tcell.ButtonNone && e.mouseDown {
			e.mouseDown = false
			if mx == e.mouseDownX && my == e.mouseDownY && inGutter {
				bufLine := e.screenToBufferLine(my - r.Y)
				if e.Folds != nil && e.Folds.FoldAt(bufLine) != nil {
					e.Folds.Toggle(bufLine)
					return EventConsumed
				}
			}
			if e.Selection != nil && e.Selection.Active {
				if e.Selection.Anchor.Line == e.Cursor.Line && e.Selection.Anchor.Col == e.Cursor.Col {
					e.Selection.Clear()
				}
			}
		}
		return EventIgnored
	}

	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)

	switch kev.Key() {
	case tcell.KeyRune, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
	default:
		if e.Undo != nil {
			e.Undo.BreakGroup()
		}
	}

	mods := kev.Modifiers()
	if mods&tcell.ModAlt != 0 || mods&tcell.ModCtrl != 0 {
		return EventIgnored
	}

	shift := mods&tcell.ModShift != 0
	hasSel := e.Selection != nil && e.Selection.Active

	multi := e.isMultiActive()

	switch kev.Key() {
	case tcell.KeyUp:
		if multi {
			e.multiMoveAll(func(cs *multicursor.CursorState) {
				if cs.Line > 0 {
					cs.Line--
					if e.Folds != nil {
						if r := e.Folds.ContainingFold(cs.Line); r != nil {
							cs.Line = r.StartLine
						}
					}
					lineLen := len([]rune(e.Buf.Lines[cs.Line]))
					if cs.Col > lineLen {
						cs.Col = lineLen
					}
				}
			})
		} else {
			e.startOrExtendSelection(shift)
			if e.Cursor.Line > 0 {
				e.Cursor.Line--
				e.skipHiddenLineUp()
				lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
				if e.Cursor.Col > lineLen {
					e.Cursor.Col = lineLen
				}
			}
		}
	case tcell.KeyDown:
		if multi {
			e.multiMoveAll(func(cs *multicursor.CursorState) {
				if cs.Line < len(e.Buf.Lines)-1 {
					cs.Line++
					if e.Folds != nil {
						if r := e.Folds.ContainingFold(cs.Line); r != nil {
							cs.Line = e.Buf.ClampLine(r.EndLine + 1)
						}
					}
					lineLen := len([]rune(e.Buf.Lines[cs.Line]))
					if cs.Col > lineLen {
						cs.Col = lineLen
					}
				}
			})
		} else {
			e.startOrExtendSelection(shift)
			if e.Cursor.Line < len(e.Buf.Lines)-1 {
				e.Cursor.Line++
				e.skipHiddenLineDown()
				lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
				if e.Cursor.Col > lineLen {
					e.Cursor.Col = lineLen
				}
			}
		}
	case tcell.KeyLeft:
		if multi {
			e.multiMoveAll(func(cs *multicursor.CursorState) {
				if cs.Col > 0 {
					cs.Col--
				} else if cs.Line > 0 {
					cs.Line--
					cs.Col = len([]rune(e.Buf.Lines[cs.Line]))
				}
			})
		} else {
			e.startOrExtendSelection(shift)
			if e.Cursor.Col > 0 {
				e.Cursor.Col--
			} else if e.Cursor.Line > 0 {
				e.Cursor.Line--
				e.Cursor.Col = len([]rune(e.Buf.Lines[e.Cursor.Line]))
			}
		}
	case tcell.KeyRight:
		if multi {
			e.multiMoveAll(func(cs *multicursor.CursorState) {
				lineLen := len([]rune(e.Buf.Lines[cs.Line]))
				if cs.Col < lineLen {
					cs.Col++
				} else if cs.Line < len(e.Buf.Lines)-1 {
					cs.Line++
					cs.Col = 0
				}
			})
		} else {
			e.startOrExtendSelection(shift)
			lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
			if e.Cursor.Col < lineLen {
				e.Cursor.Col++
			} else if e.Cursor.Line < len(e.Buf.Lines)-1 {
				e.Cursor.Line++
				e.Cursor.Col = 0
			}
		}
	case tcell.KeyHome:
		if multi {
			e.multiMoveAll(func(cs *multicursor.CursorState) {
				runes := []rune(e.Buf.Lines[cs.Line])
				firstNonSpace := 0
				for firstNonSpace < len(runes) && (runes[firstNonSpace] == ' ' || runes[firstNonSpace] == '\t') {
					firstNonSpace++
				}
				if cs.Col == firstNonSpace {
					cs.Col = 0
				} else {
					cs.Col = firstNonSpace
				}
			})
		} else {
			e.startOrExtendSelection(shift)
			e.SmartHome()
		}
	case tcell.KeyEnd:
		if multi {
			e.multiMoveAll(func(cs *multicursor.CursorState) {
				cs.Col = len([]rune(e.Buf.Lines[cs.Line]))
			})
		} else {
			e.startOrExtendSelection(shift)
			e.Cursor.Col = len([]rune(e.Buf.Lines[e.Cursor.Line]))
		}
	case tcell.KeyPgUp:
		e.startOrExtendSelection(shift)
		e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line - e.Viewport.Height)
		e.skipHiddenLineUp()
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	case tcell.KeyPgDn:
		e.startOrExtendSelection(shift)
		e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line + e.Viewport.Height)
		e.skipHiddenLineDown()
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	case tcell.KeyEnter:
		e.expandFoldAtCursor()
		if multi {
			e.multiExecEnter()
		} else {
			if hasSel {
				e.deleteSelection()
			}
			col := e.Cursor.Col
			if col < 0 {
				col = 0
			}
			lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
			if col > lineLen {
				col = lineLen
			}
			line := e.Buf.Lines[e.Cursor.Line]
			e.exec(&undo.SplitLineCommand{Line: e.Cursor.Line, Col: col})
			e.Cursor.Line++
			e.Cursor.Col = 0
			if e.AutoIndent {
				indent := leadingWhitespace(line)
				runes := []rune(line)
				charBefore := ' '
				if col > 0 && col <= len(runes) {
					charBefore = runes[col-1]
				}
				charAfter := ' '
				if col < len(runes) {
					charAfter = runes[col]
				}
				extraIndent := indentOpeners[charBefore]
				newIndent := indent
				if extraIndent {
					newIndent += e.indentUnit()
				}
				if len(newIndent) > 0 {
					e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: 0, Text: newIndent})
					e.Cursor.Col = len([]rune(newIndent))
				}
				if extraIndent && closingBrackets[charAfter] {
					e.exec(&undo.SplitLineCommand{Line: e.Cursor.Line, Col: e.Cursor.Col})
					e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line + 1, Col: 0, Text: indent})
				}
			}
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.expandFoldAtCursor()
		if multi {
			e.multiExecBackspace()
		} else {
			if hasSel {
				e.deleteSelection()
			} else if e.Cursor.Col > 0 {
				runes := []rune(e.Buf.Lines[e.Cursor.Line])
				if e.Cursor.Col > len(runes) {
					e.Cursor.Col = len(runes)
				}
				inLeadingWhitespace := true
				for i := 0; i < e.Cursor.Col && i < len(runes); i++ {
					if runes[i] != ' ' && runes[i] != '\t' {
						inLeadingWhitespace = false
						break
					}
				}
				tabSize := e.resolveTabSize()
				if inLeadingWhitespace && e.Cursor.Col > 1 && runes[e.Cursor.Col-1] == ' ' {
					target := ((e.Cursor.Col - 1) / tabSize) * tabSize
					if target == e.Cursor.Col {
						target -= tabSize
					}
					if target < 0 {
						target = 0
					}
					e.exec(&undo.DeleteSelectionCommand{
						StartLine: e.Cursor.Line, StartCol: target,
						EndLine: e.Cursor.Line, EndCol: e.Cursor.Col,
					})
					e.Cursor.Col = target
				} else {
					e.exec(&undo.DeleteRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col - 1})
					e.Cursor.Col--
				}
			} else if e.Cursor.Line > 0 {
				cmd := &undo.JoinLineCommand{Line: e.Cursor.Line}
				e.exec(cmd)
				e.Cursor.Line--
				e.Cursor.Col = cmd.PrevLen
			}
		}
	case tcell.KeyDelete:
		e.expandFoldAtCursor()
		if multi {
			e.multiExecDelete()
		} else {
			if hasSel {
				e.deleteSelection()
			} else {
				lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
				if e.Cursor.Col < lineLen {
					e.exec(&undo.DeleteRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col})
				} else if e.Cursor.Line < len(e.Buf.Lines)-1 {
					e.exec(&undo.JoinLineCommand{Line: e.Cursor.Line + 1})
				}
			}
		}
	case tcell.KeyRune:
		if kev.Modifiers() == 0 {
			e.expandFoldAtCursor()
			r := kev.Rune()
			if r != 0 {
				if multi {
					e.multiExecRune(r)
				} else if hasSel {
					start, _ := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
					e.replaceSelection(&undo.InsertRuneCommand{Line: start.Line, Col: start.Col, Rune: r})
					e.Cursor.Col = start.Col + 1
				} else {
					if e.AutoIndent && closingBrackets[r] {
						e.dedentForCloser()
					}
					e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: r})
					e.Cursor.Col++
				}
			}
		} else {
			return EventIgnored
		}
	case tcell.KeyBacktab:
		tabSize := e.resolveTabSize()
		if hasSel {
			start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
			end.Line = e.Buf.ClampLine(end.Line)
			for line := start.Line; line <= end.Line; line++ {
				remove := leadingIndentWidth(e.Buf.Lines[line], tabSize)
				if remove > 0 {
					e.exec(&undo.DeleteSelectionCommand{
						StartLine: line, StartCol: 0,
						EndLine: line, EndCol: remove,
					})
				}
			}
		} else {
			remove := leadingIndentWidth(e.Buf.Lines[e.Cursor.Line], tabSize)
			if remove > 0 {
				e.exec(&undo.DeleteSelectionCommand{
					StartLine: e.Cursor.Line, StartCol: 0,
					EndLine: e.Cursor.Line, EndCol: remove,
				})
				e.Cursor.Col -= remove
				if e.Cursor.Col < 0 {
					e.Cursor.Col = 0
				}
			}
		}
	case tcell.KeyTab:
		indent := e.indentUnit()
		if hasSel {
			start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
			for line := start.Line; line <= end.Line; line++ {
				e.exec(&undo.InsertStringCommand{Line: line, Col: 0, Text: indent})
			}
		} else {
			e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Text: indent})
			e.Cursor.Col += len([]rune(indent))
		}
	default:
		return EventIgnored
	}

	e.clampCursor()
	e.scrollViewport()
	return EventConsumed
}

func (e *EditorPaneWidget) mouseToPos(r Rect, mx, my int) (line, col int) {
	if len(e.Buf.Lines) == 0 {
		return 0, 0
	}
	gutterW := e.GutterWidth()
	screenY := my - r.Y

	if e.WordWrap && e.wrapMap != nil && screenY >= 0 && screenY < len(e.wrapMap) {
		entry := e.wrapMap[screenY]
		line = entry.bufLine
		if line >= len(e.Buf.Lines) {
			line = len(e.Buf.Lines) - 1
		}
		segVisCol := mx - r.X - gutterW
		if segVisCol < 0 {
			segVisCol = 0
		}
		segLeftCol := bufColToVisualCol(e.Buf.Lines[line], entry.startCol, e.resolveTabSize())
		col = visualColToBufCol(e.Buf.Lines[line], segLeftCol+segVisCol, e.resolveTabSize())
	} else {
		line = e.screenToBufferLine(screenY)
		visCol := mx - r.X - gutterW + e.Viewport.LeftCol
		if visCol < 0 {
			visCol = 0
		}
		if line < 0 {
			line = 0
		}
		if line >= len(e.Buf.Lines) {
			line = len(e.Buf.Lines) - 1
		}
		col = visualColToBufCol(e.Buf.Lines[line], visCol, e.resolveTabSize())
	}

	if line < 0 {
		line = 0
	}
	if line >= len(e.Buf.Lines) {
		line = len(e.Buf.Lines) - 1
	}
	lineLen := len([]rune(e.Buf.Lines[line]))
	if col > lineLen {
		col = lineLen
	}
	return
}

func (e *EditorPaneWidget) selectWord(line, col int) {
	if e.Selection == nil || line < 0 || line >= len(e.Buf.Lines) {
		return
	}
	runes := []rune(e.Buf.Lines[line])
	if len(runes) == 0 {
		return
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}
	isWord := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
	}
	start, end := col, col
	if isWord(runes[col]) {
		for start > 0 && isWord(runes[start-1]) {
			start--
		}
		for end < len(runes)-1 && isWord(runes[end+1]) {
			end++
		}
	}
	end++
	e.Selection.Start(line, start)
	e.Cursor.Line = line
	e.Cursor.Col = end
}

func (e *EditorPaneWidget) selectLine(line int) {
	if e.Selection == nil {
		return
	}
	e.Selection.Start(line, 0)
	if line < len(e.Buf.Lines)-1 {
		e.Cursor.Line = line + 1
		e.Cursor.Col = 0
	} else {
		e.Cursor.Line = line
		e.Cursor.Col = len([]rune(e.Buf.Lines[line]))
	}
}

func (e *EditorPaneWidget) resolveTabSize() int {
	if e.TabSize > 0 {
		return e.TabSize
	}
	return 4
}

func (e *EditorPaneWidget) indentUnit() string {
	if e.UseTabs {
		return "\t"
	}
	return strings.Repeat(" ", e.resolveTabSize())
}

// dedentForCloser removes one indent level from the current line when a closing
// bracket is typed on an otherwise-blank line, so `}` lands back at the
// indentation of its opening line. It is a no-op when there is non-whitespace
// before the cursor or less than a full indent unit to remove.
func (e *EditorPaneWidget) dedentForCloser() {
	runes := []rune(e.Buf.Lines[e.Cursor.Line])
	col := e.Cursor.Col
	if col > len(runes) {
		col = len(runes)
	}
	for i := 0; i < col; i++ {
		if runes[i] != ' ' && runes[i] != '\t' {
			return
		}
	}
	remove := leadingIndentWidth(e.Buf.Lines[e.Cursor.Line], e.resolveTabSize())
	if remove <= 0 || remove > col {
		return
	}
	e.exec(&undo.DeleteSelectionCommand{
		StartLine: e.Cursor.Line, StartCol: 0,
		EndLine: e.Cursor.Line, EndCol: remove,
	})
	e.Cursor.Col -= remove
	if e.Cursor.Col < 0 {
		e.Cursor.Col = 0
	}
}

type screenCell struct {
	ch     rune
	style  term.Style
	bufCol int
}

func (e *EditorPaneWidget) renderLineToScreen(line []rune, spans []highlight.Span, collapsed bool, ann []rune, tabW, leftCol, width int) []screenCell {
	cells := make([]screenCell, width)
	for i := range cells {
		cells[i] = screenCell{ch: ' ', style: term.StyleDefault, bufCol: -1}
	}
	visCol := 0
	lineLen := len(line)
	for bufCol := 0; bufCol < lineLen; bufCol++ {
		ch := line[bufCol]
		style := term.StyleDefault
		for _, sp := range spans {
			if bufCol >= sp.Start && bufCol < sp.End {
				style = sp.Style
				break
			}
		}
		if ch == '\t' {
			nextStop := ((visCol / tabW) + 1) * tabW
			for visCol < nextStop {
				sx := visCol - leftCol
				if sx >= 0 && sx < width {
					cells[sx] = screenCell{ch: ' ', style: style, bufCol: bufCol}
				}
				visCol++
			}
		} else {
			sx := visCol - leftCol
			if sx >= 0 && sx < width {
				cells[sx] = screenCell{ch: ch, style: style, bufCol: bufCol}
			}
			visCol++
		}
		if visCol-leftCol >= width {
			break
		}
	}
	if collapsed && len(ann) > 0 {
		for i, ch := range ann {
			sx := visCol + i - leftCol
			if sx >= 0 && sx < width {
				cells[sx] = screenCell{ch: ch, style: term.StyleLineNumber, bufCol: lineLen + i}
			}
		}
	}
	return cells
}

func (e *EditorPaneWidget) clampCursor() {
	e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
	if e.Cursor.Col < 0 {
		e.Cursor.Col = 0
	}
}

func (e *EditorPaneWidget) skipHiddenLineDown() {
	if e.Folds == nil {
		return
	}
	if r := e.Folds.ContainingFold(e.Cursor.Line); r != nil {
		e.Cursor.Line = e.Buf.ClampLine(r.EndLine + 1)
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	}
}

func (e *EditorPaneWidget) skipHiddenLineUp() {
	if e.Folds == nil {
		return
	}
	if r := e.Folds.ContainingFold(e.Cursor.Line); r != nil {
		e.Cursor.Line = r.StartLine
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	}
}

func (e *EditorPaneWidget) EnsureCursorVisible() {
	e.skipHiddenLineDown()
	e.scrollViewport()
}

func (e *EditorPaneWidget) ExpandFoldContaining(line int) {
	if e.Folds != nil {
		if r := e.Folds.ContainingFold(line); r != nil {
			e.Folds.Expand(r.StartLine)
		}
	}
}

func (e *EditorPaneWidget) expandFoldAtCursor() {
	if e.Folds == nil {
		return
	}
	if e.Folds.IsCollapsed(e.Cursor.Line) {
		e.Folds.Expand(e.Cursor.Line)
	}
}

func (e *EditorPaneWidget) scrollViewport() {
	if e.WordWrap {
		e.scrollViewportWrap()
		return
	}
	if e.Folds != nil && e.Folds.HasCollapsedFolds() {
		curVis := e.Folds.BufferToVisible(e.Cursor.Line)
		topVis := e.Folds.BufferToVisible(e.Viewport.TopLine)
		if curVis < 0 {
			curVis = 0
		}
		if topVis < 0 {
			topVis = 0
		}
		if curVis < topVis {
			e.Viewport.TopLine = e.Folds.VisibleToBuffer(curVis)
		}
		if curVis >= topVis+e.Viewport.Height {
			newTopVis := curVis - e.Viewport.Height + 1
			e.Viewport.TopLine = e.Folds.VisibleToBuffer(newTopVis)
		}
	} else {
		if e.Cursor.Line < e.Viewport.TopLine {
			e.Viewport.TopLine = e.Cursor.Line
		}
		if e.Cursor.Line >= e.Viewport.TopLine+e.Viewport.Height {
			e.Viewport.TopLine = e.Cursor.Line - e.Viewport.Height + 1
		}
	}
	e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
	visCol := bufColToVisualCol(e.Buf.Lines[e.Cursor.Line], e.Cursor.Col, e.resolveTabSize())
	if visCol < e.Viewport.LeftCol {
		e.Viewport.LeftCol = visCol
	}
	if visCol >= e.Viewport.LeftCol+e.Viewport.Width {
		e.Viewport.LeftCol = visCol - e.Viewport.Width + 1
	}
}

func (e *EditorPaneWidget) scrollViewportWrap() {
	e.Viewport.LeftCol = 0
	tabW := e.resolveTabSize()
	width := e.Viewport.Width
	if width < 1 {
		width = 1
	}

	curVisRow, _ := bufferPosToWrapScreenPos(e.Buf.Lines, e.Cursor.Line, e.Cursor.Col, width, tabW)
	topVisRow, _ := bufferPosToWrapScreenPos(e.Buf.Lines, e.Viewport.TopLine, 0, width, tabW)
	topVisRow += e.wrapTopOffset

	if curVisRow < topVisRow {
		e.Viewport.TopLine, e.wrapTopOffset = wrapVisualRowToTopLine(e.Buf.Lines, curVisRow, width, tabW)
	}
	if curVisRow >= topVisRow+e.Viewport.Height {
		newTop := curVisRow - e.Viewport.Height + 1
		e.Viewport.TopLine, e.wrapTopOffset = wrapVisualRowToTopLine(e.Buf.Lines, newTop, width, tabW)
	}
}

func (e *EditorPaneWidget) scrollUp(n int) {
	if e.WordWrap {
		tabW := e.resolveTabSize()
		width := e.Viewport.Width
		if width < 1 {
			width = 1
		}
		topVisRow, _ := bufferPosToWrapScreenPos(e.Buf.Lines, e.Viewport.TopLine, 0, width, tabW)
		topVisRow += e.wrapTopOffset
		newTop := topVisRow - n
		if newTop < 0 {
			newTop = 0
		}
		e.Viewport.TopLine, e.wrapTopOffset = wrapVisualRowToTopLine(e.Buf.Lines, newTop, width, tabW)
		return
	}
	if e.Folds != nil && e.Folds.HasCollapsedFolds() {
		topVis := e.Folds.BufferToVisible(e.Viewport.TopLine)
		newVis := topVis - n
		if newVis < 0 {
			newVis = 0
		}
		e.Viewport.TopLine = e.Folds.VisibleToBuffer(newVis)
	} else {
		e.Viewport.TopLine -= n
		if e.Viewport.TopLine < 0 {
			e.Viewport.TopLine = 0
		}
	}
}

func (e *EditorPaneWidget) scrollDown(n int) {
	if e.WordWrap {
		tabW := e.resolveTabSize()
		width := e.Viewport.Width
		if width < 1 {
			width = 1
		}
		topVisRow, _ := bufferPosToWrapScreenPos(e.Buf.Lines, e.Viewport.TopLine, 0, width, tabW)
		topVisRow += e.wrapTopOffset
		totalVis := totalVisualLines(e.Buf.Lines, width, tabW)
		newTop := topVisRow + n
		if newTop >= totalVis {
			newTop = totalVis - 1
		}
		if newTop < 0 {
			newTop = 0
		}
		e.Viewport.TopLine, e.wrapTopOffset = wrapVisualRowToTopLine(e.Buf.Lines, newTop, width, tabW)
		return
	}
	if e.Folds != nil && e.Folds.HasCollapsedFolds() {
		totalLines := len(e.Buf.Lines)
		visCount := e.Folds.VisibleLineCount(totalLines)
		topVis := e.Folds.BufferToVisible(e.Viewport.TopLine)
		newVis := topVis + n
		if newVis >= visCount {
			newVis = visCount - 1
		}
		if newVis < 0 {
			newVis = 0
		}
		e.Viewport.TopLine = e.Folds.VisibleToBuffer(newVis)
	} else {
		max := len(e.Buf.Lines) - 1
		if max < 0 {
			max = 0
		}
		e.Viewport.TopLine += n
		if e.Viewport.TopLine > max {
			e.Viewport.TopLine = max
		}
	}
}

func (e *EditorPaneWidget) SmartHome() {
	runes := []rune(e.Buf.Lines[e.Cursor.Line])
	firstNonSpace := 0
	for firstNonSpace < len(runes) && (runes[firstNonSpace] == ' ' || runes[firstNonSpace] == '\t') {
		firstNonSpace++
	}
	if e.Cursor.Col == firstNonSpace {
		e.Cursor.Col = 0
	} else {
		e.Cursor.Col = firstNonSpace
	}
	e.clampCursor()
	e.scrollViewport()
}
