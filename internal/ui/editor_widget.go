package ui

import (
	"ttt/internal/core/buffer"
	"ttt/internal/core/cursor"
	"ttt/internal/core/selection"
	"ttt/internal/core/undo"
	"ttt/internal/term"
	"ttt/internal/view"
	"strings"

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
	SearchQuery  string
	SearchActive int
}

func NewEditorPaneWidget(buf *buffer.Buffer, cur *cursor.Cursor, vp *view.Viewport) *EditorPaneWidget {
	return &EditorPaneWidget{
		Buf:      buf,
		Cursor:   cur,
		Viewport: vp,
	}
}

func (e *EditorPaneWidget) Focusable() bool { return true }

func (e *EditorPaneWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()

	totalLines := len(e.Buf.Lines)
	showScrollbar := totalLines > h
	editorW := w
	if showScrollbar {
		editorW = w - 1
	}

	e.Viewport.Width = editorW
	e.Viewport.Height = h

	sel := e.Selection
	hasSel := sel != nil && sel.Active

	hasSearch := e.SearchQuery != ""
	var searchMatches []FindMatch
	if hasSearch {
		searchMatches = FindInLines(e.Buf.Lines, e.SearchQuery)
	}

	for y := 0; y < h; y++ {
		lineIdx := e.Viewport.TopLine + y
		if lineIdx < totalLines {
			line := []rune(e.Buf.Lines[lineIdx])
			for x := 0; x < editorW; x++ {
				colIdx := e.Viewport.LeftCol + x
				ch := ' '
				if colIdx < len(line) {
					ch = line[colIdx]
				}
				style := term.StyleDefault
				if lineIdx == e.Cursor.Line {
					style = term.StyleActiveLine
				}
				if hasSearch {
					for mi, m := range searchMatches {
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
				if hasSel && sel.Contains(lineIdx, colIdx, e.Cursor.Line, e.Cursor.Col) {
					style = term.StyleSelection
				}
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			}
		} else {
			for x := 0; x < editorW; x++ {
				surface.SetCell(x, y, term.Cell{Ch: ' '})
			}
		}
	}

	if showScrollbar {
		thumbTop, thumbH := scrollbarThumb(totalLines, e.Viewport.TopLine, h)
		for y := 0; y < h; y++ {
			if y >= thumbTop && y < thumbTop+thumbH {
				surface.SetCell(w-1, y, term.Cell{Ch: '█', Style: term.StyleScrollbarThumb})
			} else {
				surface.SetCell(w-1, y, term.Cell{Ch: ' ', Style: term.StyleScrollbar})
			}
		}
	}

	r := e.GetRect()
	e.CursorX = e.Cursor.Col - e.Viewport.LeftCol + r.X
	e.CursorY = e.Cursor.Line - e.Viewport.TopLine + r.Y
}

func scrollbarThumb(totalLines, topLine, viewH int) (top, height int) {
	if totalLines <= viewH {
		return 0, viewH
	}
	height = viewH * viewH / totalLines
	if height < 1 {
		height = 1
	}
	scrollable := totalLines - viewH
	top = topLine * (viewH - height) / scrollable
	if top+height > viewH {
		top = viewH - height
	}
	return
}

func (e *EditorPaneWidget) exec(cmd undo.EditCommand) {
	cmd.Apply(e.Buf)
	if e.Undo != nil {
		e.Undo.Push(cmd)
	}
}

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
		indent := leadingWhitespace(e.Buf.Lines[e.Cursor.Line])
		e.exec(&undo.SplitLineCommand{Line: e.Cursor.Line, Col: col})
		e.Cursor.Line++
		e.Cursor.Col = 0
		if len(indent) > 0 {
			e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: 0, Text: indent})
			e.Cursor.Col = len([]rune(indent))
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if hasSel {
			e.deleteSelection()
		} else if e.Cursor.Col > 0 {
			e.exec(&undo.DeleteRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col - 1})
			e.Cursor.Col--
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
					e.deleteSelection()
				}
				e.exec(&undo.InsertRuneCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Rune: r})
				e.Cursor.Col++
			}
		} else {
			return EventIgnored
		}
	case tcell.KeyTab:
		if hasSel {
			e.deleteSelection()
		}
		tabSize := e.TabSize
		if tabSize <= 0 {
			tabSize = 4
		}
		e.exec(&undo.InsertStringCommand{Line: e.Cursor.Line, Col: e.Cursor.Col, Text: strings.Repeat(" ", tabSize)})
		e.Cursor.Col += tabSize
	default:
		return EventIgnored
	}

	e.clampCursor()
	e.scrollViewport()
	return EventConsumed
}

func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
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
