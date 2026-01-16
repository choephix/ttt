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
