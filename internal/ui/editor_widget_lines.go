package ui

import (
	"sort"
	"strings"

	"github.com/eugenioenko/ttt/internal/core/undo"
)

func (e *EditorPaneWidget) MoveLineUp() {
	hasSel := e.Selection != nil && e.Selection.Active
	if hasSel {
		start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
		if start.Line <= 0 {
			return
		}
		for line := start.Line; line <= end.Line; line++ {
			e.exec(&undo.SwapLineCommand{Line1: line, Line2: line - 1})
		}
		e.Cursor.Line--
		e.Selection.Anchor.Line--
	} else {
		if e.Cursor.Line <= 0 {
			return
		}
		e.exec(&undo.SwapLineCommand{Line1: e.Cursor.Line, Line2: e.Cursor.Line - 1})
		e.Cursor.Line--
	}
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) MoveLineDown() {
	hasSel := e.Selection != nil && e.Selection.Active
	if hasSel {
		start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
		if end.Line >= len(e.Buf.Lines)-1 {
			return
		}
		for line := end.Line; line >= start.Line; line-- {
			e.exec(&undo.SwapLineCommand{Line1: line, Line2: line + 1})
		}
		e.Cursor.Line++
		e.Selection.Anchor.Line++
	} else {
		if e.Cursor.Line >= len(e.Buf.Lines)-1 {
			return
		}
		e.exec(&undo.SwapLineCommand{Line1: e.Cursor.Line, Line2: e.Cursor.Line + 1})
		e.Cursor.Line++
	}
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) DuplicateLine() {
	e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
	text := e.Buf.Lines[e.Cursor.Line]
	e.exec(&undo.InsertLineCommand{Idx: e.Cursor.Line + 1, Text: text})
	e.Cursor.Line++
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) DeleteLine() {
	e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
	if len(e.Buf.Lines) <= 1 {
		e.exec(&undo.DeleteSelectionCommand{
			StartLine: 0, StartCol: 0,
			EndLine: 0, EndCol: len([]rune(e.Buf.Lines[0])),
		})
		e.Cursor.Col = 0
	} else {
		e.exec(&undo.DeleteLineCommand{Idx: e.Cursor.Line})
		e.Cursor.Line = e.Buf.ClampLine(e.Cursor.Line)
	}
	lineLen := len([]rune(e.Buf.Lines[e.Cursor.Line]))
	if e.Cursor.Col > lineLen {
		e.Cursor.Col = lineLen
	}
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) JoinLines() {
	if e.Undo != nil {
		e.Undo.BreakGroup()
	}
	hasSel := e.Selection != nil && e.Selection.Active
	if hasSel {
		start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
		endLine := end.Line
		if end.Col == 0 && endLine > start.Line {
			endLine--
		}
		// Join all lines in the selection range from the first line
		for endLine > start.Line {
			cmd := &undo.JoinNextLineCommand{Line: start.Line}
			e.exec(cmd)
			endLine--
		}
		e.Selection.Active = false
		e.Cursor.Line = start.Line
		lineLen := len([]rune(e.Buf.Lines[start.Line]))
		if e.Cursor.Col > lineLen {
			e.Cursor.Col = lineLen
		}
	} else {
		if e.Cursor.Line >= len(e.Buf.Lines)-1 {
			return
		}
		cmd := &undo.JoinNextLineCommand{Line: e.Cursor.Line}
		e.exec(cmd)
		e.Cursor.Col = cmd.JoinCol
	}
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) InsertLineBelow() {
	e.exec(&undo.InsertLineCommand{Idx: e.Cursor.Line + 1, Text: ""})
	e.Cursor.Line++
	e.Cursor.Col = 0
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) InsertLineAbove() {
	e.exec(&undo.InsertLineCommand{Idx: e.Cursor.Line, Text: ""})
	e.Cursor.Col = 0
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) commentPrefix() string {
	prefix := "//"
	if e.Highlighter != nil {
		lang := strings.ToLower(e.Highlighter.Language())
		switch lang {
		case "python", "ruby", "bash", "shell", "yaml", "toml":
			prefix = "#"
		case "lua", "sql":
			prefix = "--"
		case "html", "xml":
			prefix = "<!--"
		}
	}
	return prefix
}

func (e *EditorPaneWidget) ToggleLineComment() {
	prefix := e.commentPrefix()

	startLine, endLine := e.Cursor.Line, e.Cursor.Line
	if e.Selection != nil && e.Selection.Active {
		start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
		startLine = start.Line
		endLine = end.Line
		if end.Col == 0 && endLine > startLine {
			endLine--
		}
	}
	startLine = e.Buf.ClampLine(startLine)
	endLine = e.Buf.ClampLine(endLine)

	allCommented := true
	for l := startLine; l <= endLine; l++ {
		trimmed := strings.TrimLeft(e.Buf.Lines[l], " \t")
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, prefix) {
			allCommented = false
			break
		}
	}

	var cmds []undo.EditCommand
	cursorDelta := 0

	for l := startLine; l <= endLine; l++ {
		runes := []rune(e.Buf.Lines[l])
		trimmed := strings.TrimLeft(string(runes), " \t")
		if trimmed == "" {
			continue
		}
		indent := len(runes) - len([]rune(trimmed))

		if allCommented {
			removeLen := len([]rune(prefix))
			if indent+removeLen < len(runes) && runes[indent+removeLen] == ' ' {
				removeLen++
			}
			cmd := &undo.DeleteSelectionCommand{
				StartLine: l, StartCol: indent,
				EndLine: l, EndCol: indent + removeLen,
			}
			cmd.Apply(e.Buf)
			cmds = append(cmds, cmd)
			if l == e.Cursor.Line {
				cursorDelta = -removeLen
			}
		} else {
			cmd := &undo.InsertStringCommand{Line: l, Col: indent, Text: prefix + " "}
			cmd.Apply(e.Buf)
			cmds = append(cmds, cmd)
			if l == e.Cursor.Line {
				cursorDelta = len([]rune(prefix)) + 1
			}
		}
	}

	if len(cmds) > 0 && e.Undo != nil {
		e.Undo.Push(&undo.BatchCommand{Commands: cmds})
		e.bufferDirty = true
	}

	e.Cursor.Col += cursorDelta
	if e.Cursor.Col < 0 {
		e.Cursor.Col = 0
	}
	e.clampCursor()
	e.scrollViewport()
}

// lineRange returns the start and end line indices for the current selection,
// or the full buffer range if no selection is active.
func (e *EditorPaneWidget) lineRange() (int, int) {
	if e.Selection != nil && e.Selection.Active {
		start, end := e.Selection.Range(e.Cursor.Line, e.Cursor.Col)
		endLine := end.Line
		if end.Col == 0 && endLine > start.Line {
			endLine--
		}
		return e.Buf.ClampLine(start.Line), e.Buf.ClampLine(endLine)
	}
	end := len(e.Buf.Lines) - 1
	if end > 0 && e.Buf.Lines[end] == "" {
		end--
	}
	return 0, end
}

// copyLines returns a copy of the buffer lines in the given range (inclusive).
func (e *EditorPaneWidget) copyLines(start, end int) []string {
	lines := make([]string, end-start+1)
	copy(lines, e.Buf.Lines[start:end+1])
	return lines
}

func (e *EditorPaneWidget) SortLinesAsc() {
	start, end := e.lineRange()
	old := e.copyLines(start, end)
	sorted := make([]string, len(old))
	copy(sorted, old)
	sort.Strings(sorted)
	e.exec(&undo.ReplaceLinesCommand{Start: start, OldLines: old, NewLines: sorted})
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) SortLinesDesc() {
	start, end := e.lineRange()
	old := e.copyLines(start, end)
	sorted := make([]string, len(old))
	copy(sorted, old)
	sort.Sort(sort.Reverse(sort.StringSlice(sorted)))
	e.exec(&undo.ReplaceLinesCommand{Start: start, OldLines: old, NewLines: sorted})
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) ReverseLines() {
	start, end := e.lineRange()
	old := e.copyLines(start, end)
	reversed := make([]string, len(old))
	for i, line := range old {
		reversed[len(old)-1-i] = line
	}
	e.exec(&undo.ReplaceLinesCommand{Start: start, OldLines: old, NewLines: reversed})
	e.clampCursor()
	e.scrollViewport()
}

func (e *EditorPaneWidget) UniqueLines() {
	start, end := e.lineRange()
	old := e.copyLines(start, end)
	seen := make(map[string]bool)
	var unique []string
	for _, line := range old {
		if !seen[line] {
			seen[line] = true
			unique = append(unique, line)
		}
	}
	e.exec(&undo.ReplaceLinesCommand{Start: start, OldLines: old, NewLines: unique})
	e.clampCursor()
	e.scrollViewport()
}
