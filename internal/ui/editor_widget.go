package ui

import (
	"macro/internal/core/buffer"
	"macro/internal/core/cursor"
	"macro/internal/term"
	"macro/internal/view"

	"github.com/gdamore/tcell/v2"
)

type EditorPaneWidget struct {
	BaseWidget
	Buf      *buffer.Buffer
	Cursor   *cursor.Cursor
	Viewport *view.Viewport
	CursorX  int
	CursorY  int
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
	e.Viewport.Width = w
	e.Viewport.Height = h

	for y := 0; y < h; y++ {
		lineIdx := e.Viewport.TopLine + y
		if lineIdx < len(e.Buf.Lines) {
			line := []rune(e.Buf.Lines[lineIdx])
			for x := 0; x < w; x++ {
				colIdx := e.Viewport.LeftCol + x
				ch := ' '
				if colIdx < len(line) {
					ch = line[colIdx]
				}
				surface.SetCell(x, y, term.Cell{Ch: ch})
			}
		} else {
			surface.SetCell(0, y, term.Cell{Ch: '~', Style: term.StyleLineNumber})
			for x := 1; x < w; x++ {
				surface.SetCell(x, y, term.Cell{Ch: ' '})
			}
		}
	}

	r := e.GetRect()
	e.CursorX = e.Cursor.Col - e.Viewport.LeftCol + r.X
	e.CursorY = e.Cursor.Line - e.Viewport.TopLine + r.Y
}

func (e *EditorPaneWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventIgnored
	}

	switch kev.Key() {
	case tcell.KeyUp:
		if e.Cursor.Line > 0 {
			e.Cursor.Line--
			lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
			if e.Cursor.Col > lineLen {
				e.Cursor.Col = lineLen
			}
		}
	case tcell.KeyDown:
		if e.Cursor.Line < len(e.Buf.Lines)-1 {
			e.Cursor.Line++
			lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
			if e.Cursor.Col > lineLen {
				e.Cursor.Col = lineLen
			}
		}
	case tcell.KeyLeft:
		if e.Cursor.Col > 0 {
			e.Cursor.Col--
		} else if e.Cursor.Line > 0 {
			e.Cursor.Line--
			e.Cursor.Col = len([]rune(e.Buf.Lines[e.Cursor.Line]))
		}
	case tcell.KeyRight:
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col < lineLen {
			e.Cursor.Col++
		} else if e.Cursor.Line < len(e.Buf.Lines)-1 {
			e.Cursor.Line++
			e.Cursor.Col = 0
		}
	case tcell.KeyEnter:
		line := []rune(e.Buf.Lines[e.Cursor.Line])
		if e.Cursor.Col < 0 {
			e.Cursor.Col = 0
		}
		if e.Cursor.Col > len(line) {
			e.Cursor.Col = len(line)
		}
		left := string(line[:e.Cursor.Col])
		right := string(line[e.Cursor.Col:])
		e.Buf.Lines[e.Cursor.Line] = left
		e.Buf.InsertLine(e.Cursor.Line+1, right)
		e.Cursor.Line++
		e.Cursor.Col = 0
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.Cursor.Col > 0 {
			e.Buf.DeleteRune(e.Cursor.Line, e.Cursor.Col-1)
			e.Cursor.Col--
		} else if e.Cursor.Line > 0 {
			prevLen := len([]rune(e.Buf.Lines[e.Cursor.Line-1]))
			e.Buf.Lines[e.Cursor.Line-1] += e.Buf.Lines[e.Cursor.Line]
			e.Buf.DeleteLine(e.Cursor.Line)
			e.Cursor.Line--
			e.Cursor.Col = prevLen
		}
	case tcell.KeyHome:
		e.Cursor.Col = 0
	case tcell.KeyEnd:
		e.Cursor.Col = len([]rune(e.Buf.Lines[e.Cursor.Line]))
	case tcell.KeyPgUp:
		e.Cursor.Line -= e.Viewport.Height
		if e.Cursor.Line < 0 {
			e.Cursor.Line = 0
		}
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	case tcell.KeyPgDn:
		e.Cursor.Line += e.Viewport.Height
		if e.Cursor.Line >= len(e.Buf.Lines) {
			e.Cursor.Line = len(e.Buf.Lines) - 1
		}
		lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	case tcell.KeyRune:
		if kev.Modifiers() == 0 {
			r := kev.Rune()
			if r != 0 {
				e.Buf.InsertRune(e.Cursor.Line, e.Cursor.Col, r)
				e.Cursor.Col++
			}
		} else {
			return EventIgnored
		}
	case tcell.KeyTab:
		for i := 0; i < 4; i++ {
			e.Buf.InsertRune(e.Cursor.Line, e.Cursor.Col, ' ')
			e.Cursor.Col++
		}
	default:
		return EventIgnored
	}

	e.clampCursor()
	e.scrollViewport()
	return EventConsumed
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
