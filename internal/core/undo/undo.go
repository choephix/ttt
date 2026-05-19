package undo

import "macro/internal/core/buffer"

// EditCommand defines the interface for undoable buffer edits.
type EditCommand interface {
	Apply(b *buffer.Buffer)
	Undo(b *buffer.Buffer)
}

// UndoStack manages undo and redo stacks.
type UndoStack struct {
	undo []EditCommand
	redo []EditCommand
}

// Push adds a command to the undo stack and clears the redo stack.
func (s *UndoStack) Push(cmd EditCommand) {
	s.undo = append(s.undo, cmd)
	s.redo = nil
}

// Undo undoes the last command.
func (s *UndoStack) Undo(b *buffer.Buffer) {
	if len(s.undo) == 0 {
		return
	}
	cmd := s.undo[len(s.undo)-1]
	s.undo = s.undo[:len(s.undo)-1]
	cmd.Undo(b)
	s.redo = append(s.redo, cmd)
}

// Redo re-applies the last undone command.
func (s *UndoStack) Redo(b *buffer.Buffer) {
	if len(s.redo) == 0 {
		return
	}
	cmd := s.redo[len(s.redo)-1]
	s.redo = s.redo[:len(s.redo)-1]
	cmd.Apply(b)
	s.undo = append(s.undo, cmd)
}

// InsertRuneCommand implements EditCommand for inserting a rune.
type InsertRuneCommand struct {
	Line, Col int
	Rune      rune
}

func (c *InsertRuneCommand) Apply(b *buffer.Buffer) {
	b.InsertRune(c.Line, c.Col, c.Rune)
}

func (c *InsertRuneCommand) Undo(b *buffer.Buffer) {
	b.DeleteRune(c.Line, c.Col)
}

// DeleteRangeCommand implements EditCommand for deleting a range of runes.
type DeleteRangeCommand struct {
	Line, Start, End int
	Deleted          string
}

func (c *DeleteRangeCommand) Apply(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	l := []rune(b.Lines[c.Line])
	if c.Start < 0 || c.End > len(l) || c.Start >= c.End {
		return
	}
	c.Deleted = string(l[c.Start:c.End])
	b.Lines[c.Line] = string(append(l[:c.Start], l[c.End:]...))
	b.Dirty = true
}

func (c *DeleteRangeCommand) Undo(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	l := []rune(b.Lines[c.Line])
	if c.Start < 0 || c.Start > len(l) {
		return
	}
	// Insert the deleted text back at the original position
	newRunes := append([]rune{}, l[:c.Start]...)
	newRunes = append(newRunes, []rune(c.Deleted)...)
	newRunes = append(newRunes, l[c.Start:]...)
	b.Lines[c.Line] = string(newRunes)
	b.Dirty = true
}

// DeleteRuneCommand implements EditCommand for deleting a rune.
type DeleteRuneCommand struct {
	Line, Col int
	Rune      rune
}

func (c *DeleteRuneCommand) Apply(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	runes := []rune(b.Lines[c.Line])
	if c.Col < 0 || c.Col >= len(runes) {
		return
	}
	c.Rune = runes[c.Col]
	b.DeleteRune(c.Line, c.Col)
}

func (c *DeleteRuneCommand) Undo(b *buffer.Buffer) {
	b.InsertRune(c.Line, c.Col, c.Rune)
}

// SplitLineCommand implements EditCommand for splitting a line (Enter key).
type SplitLineCommand struct {
	Line, Col int
}

func (c *SplitLineCommand) Apply(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	line := []rune(b.Lines[c.Line])
	col := c.Col
	if col > len(line) {
		col = len(line)
	}
	left := string(line[:col])
	right := string(line[col:])
	b.Lines[c.Line] = left
	b.InsertLine(c.Line+1, right)
}

func (c *SplitLineCommand) Undo(b *buffer.Buffer) {
	if c.Line+1 >= len(b.Lines) {
		return
	}
	b.Lines[c.Line] += b.Lines[c.Line+1]
	b.DeleteLine(c.Line + 1)
	b.Dirty = true
}

// JoinLineCommand implements EditCommand for joining a line with the previous one (Backspace at col 0).
type JoinLineCommand struct {
	Line    int
	PrevLen int
}

func (c *JoinLineCommand) Apply(b *buffer.Buffer) {
	if c.Line <= 0 || c.Line >= len(b.Lines) {
		return
	}
	c.PrevLen = len([]rune(b.Lines[c.Line-1]))
	b.Lines[c.Line-1] += b.Lines[c.Line]
	b.DeleteLine(c.Line)
	b.Dirty = true
}

func (c *JoinLineCommand) Undo(b *buffer.Buffer) {
	if c.Line-1 < 0 || c.Line-1 >= len(b.Lines) {
		return
	}
	combined := []rune(b.Lines[c.Line-1])
	left := string(combined[:c.PrevLen])
	right := string(combined[c.PrevLen:])
	b.Lines[c.Line-1] = left
	b.InsertLine(c.Line, right)
}

// InsertLineCommand implements EditCommand for inserting a line.
type InsertLineCommand struct {
	Idx  int
	Text string
}

func (c *InsertLineCommand) Apply(b *buffer.Buffer) {
	b.InsertLine(c.Idx, c.Text)
}

func (c *InsertLineCommand) Undo(b *buffer.Buffer) {
	b.DeleteLine(c.Idx)
}

// InsertStringCommand implements EditCommand for inserting multiple runes (e.g. tab spaces).
type InsertStringCommand struct {
	Line, Col int
	Text      string
}

func (c *InsertStringCommand) Apply(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	runes := []rune(b.Lines[c.Line])
	col := c.Col
	if col > len(runes) {
		col = len(runes)
	}
	insert := []rune(c.Text)
	newRunes := append([]rune{}, runes[:col]...)
	newRunes = append(newRunes, insert...)
	newRunes = append(newRunes, runes[col:]...)
	b.Lines[c.Line] = string(newRunes)
	b.Dirty = true
}

func (c *InsertStringCommand) Undo(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	runes := []rune(b.Lines[c.Line])
	tLen := len([]rune(c.Text))
	col := c.Col
	if col+tLen > len(runes) {
		return
	}
	newRunes := append(runes[:col], runes[col+tLen:]...)
	b.Lines[c.Line] = string(newRunes)
	b.Dirty = true
}

// DeleteSelectionCommand deletes a multi-line range and stores the deleted text for undo.
type DeleteSelectionCommand struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	Deleted             string
}

func (c *DeleteSelectionCommand) Apply(b *buffer.Buffer) {
	if c.StartLine >= len(b.Lines) {
		return
	}

	startRunes := []rune(b.Lines[c.StartLine])
	sc := c.StartCol
	if sc > len(startRunes) {
		sc = len(startRunes)
	}

	if c.StartLine == c.EndLine {
		endRunes := []rune(b.Lines[c.StartLine])
		ec := c.EndCol
		if ec > len(endRunes) {
			ec = len(endRunes)
		}
		c.Deleted = string(endRunes[sc:ec])
		b.Lines[c.StartLine] = string(startRunes[:sc]) + string(endRunes[ec:])
		b.Dirty = true
		return
	}

	endRunes := []rune(b.Lines[c.EndLine])
	ec := c.EndCol
	if ec > len(endRunes) {
		ec = len(endRunes)
	}

	// Build deleted text
	var del []rune
	del = append(del, startRunes[sc:]...)
	del = append(del, '\n')
	for l := c.StartLine + 1; l < c.EndLine; l++ {
		del = append(del, []rune(b.Lines[l])...)
		del = append(del, '\n')
	}
	del = append(del, endRunes[:ec]...)
	c.Deleted = string(del)

	// Merge start prefix with end suffix
	b.Lines[c.StartLine] = string(startRunes[:sc]) + string(endRunes[ec:])

	// Remove lines between
	if c.EndLine > c.StartLine {
		b.Lines = append(b.Lines[:c.StartLine+1], b.Lines[c.EndLine+1:]...)
	}
	b.Dirty = true
}

func (c *DeleteSelectionCommand) Undo(b *buffer.Buffer) {
	if c.StartLine >= len(b.Lines) {
		return
	}

	// Split current merged line at StartCol
	runes := []rune(b.Lines[c.StartLine])
	sc := c.StartCol
	if sc > len(runes) {
		sc = len(runes)
	}
	prefix := string(runes[:sc])
	suffix := string(runes[sc:])

	// Re-insert deleted text
	delLines := splitLines(c.Deleted)
	if len(delLines) == 1 {
		b.Lines[c.StartLine] = prefix + delLines[0] + suffix
	} else {
		newLines := make([]string, 0, len(b.Lines)+len(delLines)-1)
		newLines = append(newLines, b.Lines[:c.StartLine]...)
		newLines = append(newLines, prefix+delLines[0])
		for i := 1; i < len(delLines)-1; i++ {
			newLines = append(newLines, delLines[i])
		}
		newLines = append(newLines, delLines[len(delLines)-1]+suffix)
		newLines = append(newLines, b.Lines[c.StartLine+1:]...)
		b.Lines = newLines
	}
	b.Dirty = true
}

type PasteCommand struct {
	Line, Col int
	Text      string
	Suffix    string
}

func (c *PasteCommand) Apply(b *buffer.Buffer) {
	if c.Line < 0 || c.Line >= len(b.Lines) {
		return
	}
	lines := splitLines(c.Text)
	currentRunes := []rune(b.Lines[c.Line])
	col := c.Col
	if col > len(currentRunes) {
		col = len(currentRunes)
	}
	prefix := string(currentRunes[:col])

	if len(lines) == 1 {
		b.Lines[c.Line] = prefix + lines[0] + c.Suffix
	} else {
		newLines := make([]string, 0, len(b.Lines)+len(lines)-1)
		newLines = append(newLines, b.Lines[:c.Line]...)
		newLines = append(newLines, prefix+lines[0])
		for i := 1; i < len(lines)-1; i++ {
			newLines = append(newLines, lines[i])
		}
		newLines = append(newLines, lines[len(lines)-1]+c.Suffix)
		newLines = append(newLines, b.Lines[c.Line+1:]...)
		b.Lines = newLines
	}
	b.Dirty = true
}

func (c *PasteCommand) Undo(b *buffer.Buffer) {
	lines := splitLines(c.Text)
	if len(lines) == 1 {
		if c.Line < 0 || c.Line >= len(b.Lines) {
			return
		}
		runes := []rune(b.Lines[c.Line])
		tLen := len([]rune(lines[0]))
		col := c.Col
		if col+tLen > len(runes) {
			return
		}
		newRunes := append(runes[:col], runes[col+tLen:]...)
		b.Lines[c.Line] = string(newRunes)
	} else {
		if c.Line < 0 || c.Line >= len(b.Lines) {
			return
		}
		runes := []rune(b.Lines[c.Line])
		col := c.Col
		if col > len(runes) {
			col = len(runes)
		}
		restored := string(runes[:col]) + c.Suffix
		endLine := c.Line + len(lines) - 1
		if endLine >= len(b.Lines) {
			endLine = len(b.Lines) - 1
		}
		newLines := make([]string, 0, len(b.Lines)-(endLine-c.Line))
		newLines = append(newLines, b.Lines[:c.Line]...)
		newLines = append(newLines, restored)
		if endLine+1 < len(b.Lines) {
			newLines = append(newLines, b.Lines[endLine+1:]...)
		}
		b.Lines = newLines
	}
	b.Dirty = true
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, ch := range s {
		if ch == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
