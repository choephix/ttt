package ui

import (
	"strconv"
	"strings"
	"time"
	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/core/cursor"
	"github.com/eugenioenko/ttt/internal/core/highlight"
	"github.com/eugenioenko/ttt/internal/core/selection"
	"github.com/eugenioenko/ttt/internal/core/undo"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/view"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type EditorPaneWidget struct {
	BaseWidget
	Buf          *buffer.Buffer
	Cursor       *cursor.Cursor
	Viewport     *view.Viewport
	Undo         *undo.UndoStack
	Selection    *selection.Selection
	CursorX      int
	CursorY      int
	TabSize      int
	LineNumbers  bool
	Highlighter  *highlight.Highlighter
	SearchQuery   string
	SearchMatches []FindMatch
	SearchActive  int
	lastClickTime int64
	lastClickLine int
	lastClickCol  int
	clickCount    int
	mouseDown     bool
	scrollbar     Scrollbar
	OnChange      func()
}

func NewEditorPaneWidget(buf *buffer.Buffer, cur *cursor.Cursor, vp *view.Viewport) *EditorPaneWidget {
	return &EditorPaneWidget{
		Buf:      buf,
		Cursor:   cur,
		Viewport: vp,
	}
}

func (e *EditorPaneWidget) Focusable() bool { return true }

func (e *EditorPaneWidget) gutterWidth() int {
	if !e.LineNumbers {
		return 0
	}
	digits := len(strconv.Itoa(len(e.Buf.Lines)))
	if digits < 2 {
		digits = 2
	}
	return digits + 3
}

func (e *EditorPaneWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()

	totalLines := len(e.Buf.Lines)
	showScrollbar := totalLines > h
	gutterW := e.gutterWidth()
	editorW := w - gutterW
	if showScrollbar {
		editorW--
	}
	if editorW < 1 {
		editorW = 1
	}

	e.Viewport.Width = editorW
	e.Viewport.Height = h

	sel := e.Selection
	hasSel := sel != nil && sel.Active

	hasSearch := len(e.SearchMatches) > 0

	matchLine, matchCol, hasMatch := e.findMatchingBracket()

	if e.Viewport.TopLine < 0 {
		e.Viewport.TopLine = 0
	}
	for y := 0; y < h; y++ {
		lineIdx := e.Viewport.TopLine + y

		if gutterW > 0 {
			gutterStyle := term.StyleLineNumber
			if lineIdx < totalLines && lineIdx == e.Cursor.Line {
				gutterStyle = term.StyleActiveLine
			}
			numStr := ""
			if lineIdx < totalLines {
				numStr = strconv.Itoa(lineIdx + 1)
			}
			padded := " " + strings.Repeat(" ", gutterW-3-len(numStr)) + numStr + "  "
			for i, ch := range padded {
				surface.SetCell(i, y, term.Cell{Ch: ch, Style: gutterStyle})
			}
		}

		if lineIdx < totalLines {
			line := []rune(e.Buf.Lines[lineIdx])
			var syntaxSpans []highlight.Span
			if e.Highlighter != nil {
				syntaxSpans = e.Highlighter.HighlightLine(e.Buf.Lines[lineIdx])
			}
			for x := 0; x < editorW; x++ {
				colIdx := e.Viewport.LeftCol + x
				ch := ' '
				if colIdx < len(line) {
					ch = line[colIdx]
				}
				style := term.StyleDefault
				for _, sp := range syntaxSpans {
					if colIdx >= sp.Start && colIdx < sp.End {
						style = sp.Style
						break
					}
				}
				if hasSearch {
					for mi, m := range e.SearchMatches {
						if m.Line == lineIdx && colIdx >= m.Col && colIdx < m.Col+m.Len {
							if mi == e.SearchActive {
								style = term.StyleSearchActive
							} else {
								style = term.StyleSearchMatch
							}
							break
						}
					}
				}
				isSearchHighlight := style == term.StyleSearchActive || style == term.StyleSearchMatch
				bgStyle := term.Style(0)
				if hasSel && sel.Contains(lineIdx, colIdx, e.Cursor.Line, e.Cursor.Col) {
					bgStyle = term.StyleSelection
				} else if lineIdx == e.Cursor.Line && !hasSel && !isSearchHighlight {
					bgStyle = term.StyleActiveLine
				}
				if hasMatch && ((lineIdx == e.Cursor.Line && colIdx == e.Cursor.Col) ||
					(lineIdx == matchLine && colIdx == matchCol)) {
					bgStyle = term.StyleBracketMatch
				}
				surface.SetCell(gutterW+x, y, term.Cell{Ch: ch, Style: style, BgStyle: bgStyle})
			}
		} else {
			for x := 0; x < editorW; x++ {
				surface.SetCell(gutterW+x, y, term.Cell{Ch: ' '})
			}
		}
	}

	if showScrollbar {
		r := e.GetRect()
		e.scrollbar.X = r.X + w - 1
		e.scrollbar.Y = r.Y
		e.scrollbar.Height = h
		e.scrollbar.TotalItems = totalLines
		e.scrollbar.TopItem = e.Viewport.TopLine
		e.scrollbar.Render(surface, w-1, 0)
	}

	r := e.GetRect()
	e.CursorX = e.Cursor.Col - e.Viewport.LeftCol + gutterW + r.X
	e.CursorY = e.Cursor.Line - e.Viewport.TopLine + r.Y
}


func (e *EditorPaneWidget) exec(cmd undo.EditCommand) {
	cmd.Apply(e.Buf)
	if e.Undo != nil {
		e.Undo.Push(cmd)
	}
	if e.OnChange != nil {
		e.OnChange()
	}
}

func (e *EditorPaneWidget) ExecCommand(cmd undo.EditCommand) { e.exec(cmd) }

func (e *EditorPaneWidget) deleteSelection() {
	if e.Selection == nil || !e.Selection.Active {
		return
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
			e.Viewport.TopLine = newTop
			return EventConsumed
		}
		if e.scrollbar.IsDragging() {
			return EventConsumed
		}

		if btn&tcell.WheelUp != 0 {
			e.Viewport.TopLine -= 3
			if e.Viewport.TopLine < 0 {
				e.Viewport.TopLine = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			max := len(e.Buf.Lines) - e.Viewport.Height
			if max < 0 {
				max = 0
			}
			e.Viewport.TopLine += 3
			if e.Viewport.TopLine > max {
				e.Viewport.TopLine = max
			}
			return EventConsumed
		}
		if btn&tcell.Button1 != 0 {
			r := e.GetRect()
			mx, my := mev.Position()
			line, col := e.mouseToPos(r, mx, my)

			if !e.mouseDown {
				e.mouseDown = true

				now := time.Now().UnixMilli()
				if now-e.lastClickTime < 400 && line == e.lastClickLine && col == e.lastClickCol {
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
			return EventConsumed
		}
		if btn == tcell.ButtonNone && e.mouseDown {
			e.mouseDown = false
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

	shift := kev.Modifiers()&tcell.ModShift != 0
	hasSel := e.Selection != nil && e.Selection.Active

	switch kev.Key() {
	case tcell.KeyUp:
		e.startOrExtendSelection(shift)
		if e.Cursor.Line > 0 {
			e.Cursor.Line--
			lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
			if e.Cursor.Col > lineLen {
				e.Cursor.Col = lineLen
			}
		}
	case tcell.KeyDown:
		e.startOrExtendSelection(shift)
		if e.Cursor.Line < len(e.Buf.Lines)-1 {
			e.Cursor.Line++
			lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
			if e.Cursor.Col > lineLen {
				e.Cursor.Col = lineLen
			}
		}
	case tcell.KeyLeft:
		e.startOrExtendSelection(shift)
		if e.Cursor.Col > 0 {
			e.Cursor.Col--
		} else if e.Cursor.Line > 0 {
			e.Cursor.Line--
			e.Cursor.Col = len([]rune(e.Buf.Lines[e.Cursor.Line]))
		}
	case tcell.KeyRight:
		e.startOrExtendSelection(shift)
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col < lineLen {
			e.Cursor.Col++
		} else if e.Cursor.Line < len(e.Buf.Lines)-1 {
			e.Cursor.Line++
			e.Cursor.Col = 0
		}
	case tcell.KeyHome:
		e.startOrExtendSelection(shift)
		e.Cursor.Col = 0
	case tcell.KeyEnd:
		e.startOrExtendSelection(shift)
		e.Cursor.Col = len([]rune(e.Buf.Lines[e.Cursor.Line]))
	case tcell.KeyPgUp:
		e.startOrExtendSelection(shift)
		e.Cursor.Line -= e.Viewport.Height
		if e.Cursor.Line < 0 {
			e.Cursor.Line = 0
		}
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	case tcell.KeyPgDn:
		e.startOrExtendSelection(shift)
		e.Cursor.Line += e.Viewport.Height
		if e.Cursor.Line >= len(e.Buf.Lines) {
			e.Cursor.Line = len(e.Buf.Lines) - 1
		}
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	case tcell.KeyEnter:
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
		extraIndent := charBefore == '{' || charBefore == '(' || charBefore == '[' || charBefore == ':'
		e.exec(&undo.SplitLineCommand{Line: e.Cursor.Line, Col: col})
		e.Cursor.Line++
		e.Cursor.Col = 0
		tabSize := e.TabSize
		if tabSize <= 0 {
			tabSize = 4
		}
		newIndent := indent
		if extraIndent {
			newIndent += strings.Repeat(" ", tabSize)
		}
		if len(newIndent) > 0 {
			e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: 0, Text: newIndent})
			e.Cursor.Col = len([]rune(newIndent))
		}
		if extraIndent && (charAfter == '}' || charAfter == ')' || charAfter == ']') {
			e.exec(&undo.SplitLineCommand{Line: e.Cursor.Line, Col: e.Cursor.Col})
			e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line + 1, Col: 0, Text: indent})
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if hasSel {
			e.deleteSelection()
		} else if e.Cursor.Col > 0 {
			runes := []rune(e.Buf.Lines[e.Cursor.Line])
			inLeadingWhitespace := true
			for i := 0; i < e.Cursor.Col && i < len(runes); i++ {
				if runes[i] != ' ' && runes[i] != '\t' {
					inLeadingWhitespace = false
					break
				}
			}
			tabSize := e.TabSize
			if tabSize <= 0 {
				tabSize = 4
			}
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
	case tcell.KeyDelete:
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
	case tcell.KeyRune:
		if kev.Modifiers() == 0 {
			r := kev.Rune()
			if r != 0 {
				if hasSel {
					if closing, ok := autoPairs[r]; ok {
						e.deleteSelection()
						e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: r})
						e.Cursor.Col++
						e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: closing})
					} else {
						e.deleteSelection()
						e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: r})
						e.Cursor.Col++
					}
				} else if closing, skip := autoCloseSkip[r]; skip && e.charAtCursor() == r {
					_ = closing
					e.Cursor.Col++
				} else if closing, ok := autoPairs[r]; ok {
					e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: r})
					e.Cursor.Col++
					e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: closing})
				} else {
					e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: r})
					e.Cursor.Col++
				}
			}
		} else {
			return EventIgnored
		}
	case tcell.KeyBacktab:
		if hasSel {
			start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
			tabSize := e.TabSize
			if tabSize <= 0 {
				tabSize = 4
			}
			for line := start.Line; line <= end.Line; line++ {
				runes := []rune(e.Buf.Lines[line])
				remove := 0
				for remove < tabSize && remove < len(runes) && runes[remove] == ' ' {
					remove++
				}
				if remove > 0 {
					e.exec(&undo.DeleteSelectionCommand{
						StartLine: line, StartCol: 0,
						EndLine: line, EndCol: remove,
					})
				}
			}
		}
	case tcell.KeyTab:
		tabSize := e.TabSize
		if tabSize <= 0 {
			tabSize = 4
		}
		if hasSel {
			start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
			indent := strings.Repeat(" ", tabSize)
			for line := start.Line; line <= end.Line; line++ {
				e.exec(&undo.InsertStringCommand{Line: line, Col: 0, Text: indent})
			}
		} else {
			e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Text: strings.Repeat(" ", tabSize)})
			e.Cursor.Col += tabSize
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
	gutterW := e.gutterWidth()
	line = my - r.Y + e.Viewport.TopLine
	col = mx - r.X - gutterW + e.Viewport.LeftCol
	if col < 0 {
		col = 0
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
	if e.Selection == nil {
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

func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}

var autoPairs = map[rune]rune{
	'(': ')', '{': '}', '[': ']',
	'"': '"', '\'': '\'', '`': '`',
}

var autoCloseSkip = map[rune]bool{
	')': true, '}': true, ']': true,
	'"': true, '\'': true, '`': true,
}

func (e *EditorPaneWidget) charAtCursor() rune {
	if e.Cursor.Line >= len(e.Buf.Lines) {
		return 0
	}
	runes := []rune(e.Buf.Lines[e.Cursor.Line])
	if e.Cursor.Col >= len(runes) {
		return 0
	}
	return runes[e.Cursor.Col]
}

var bracketPairs = map[rune]rune{
	'(': ')', ')': '(',
	'[': ']', ']': '[',
	'{': '}', '}': '{',
}

var closingBrackets = map[rune]bool{')': true, ']': true, '}': true}

func (e *EditorPaneWidget) findMatchingBracket() (int, int, bool) {
	if e.Cursor.Line < 0 || e.Cursor.Line >= len(e.Buf.Lines) {
		return 0, 0, false
	}
	runes := []rune(e.Buf.Lines[e.Cursor.Line])
	if e.Cursor.Col < 0 || e.Cursor.Col >= len(runes) {
		return 0, 0, false
	}
	ch := runes[e.Cursor.Col]
	match, ok := bracketPairs[ch]
	if !ok {
		return 0, 0, false
	}
	dir := 1
	if closingBrackets[ch] {
		dir = -1
	}
	depth := 1
	line, col := e.Cursor.Line, e.Cursor.Col
	for {
		col += dir
		lr := []rune(e.Buf.Lines[line])
		if col < 0 {
			line--
			if line < 0 {
				return 0, 0, false
			}
			lr = []rune(e.Buf.Lines[line])
			col = len(lr) - 1
			if col < 0 {
				col = 0
				continue
			}
		} else if col >= len(lr) {
			line++
			if line >= len(e.Buf.Lines) {
				return 0, 0, false
			}
			col = 0
			lr = []rune(e.Buf.Lines[line])
			if len(lr) == 0 {
				continue
			}
		}
		c := lr[col]
		if c == ch {
			depth++
		} else if c == match {
			depth--
			if depth == 0 {
				return line, col, true
			}
		}
	}
}

func (e *EditorPaneWidget) clampCursor() {
	if e.Cursor.Line < 0 {
		e.Cursor.Line = 0
	}
	if e.Cursor.Line >= len(e.Buf.Lines) {
		e.Cursor.Line = len(e.Buf.Lines) - 1
	}
	if e.Cursor.Col < 0 {
		e.Cursor.Col = 0
	}
}

func (e *EditorPaneWidget) scrollViewport() {
	if e.Cursor.Line < e.Viewport.TopLine {
		e.Viewport.TopLine = e.Cursor.Line
	}
	if e.Cursor.Line >= e.Viewport.TopLine+e.Viewport.Height {
		e.Viewport.TopLine = e.Cursor.Line - e.Viewport.Height + 1
	}
	if e.Cursor.Col < e.Viewport.LeftCol {
		e.Viewport.LeftCol = e.Cursor.Col
	}
	if e.Cursor.Col >= e.Viewport.LeftCol+e.Viewport.Width {
		e.Viewport.LeftCol = e.Cursor.Col - e.Viewport.Width + 1
	}
}
