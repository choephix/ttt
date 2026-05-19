package view

import (
	"fmt"
)

// StatusBar holds information to display in the status bar.
type StatusBar struct {
	FileName string
	Line     int
	Col      int
	Dirty    bool
}

// RenderStatusBar returns a string representing the status bar.
func (s *StatusBar) RenderStatusBar(width int) string {
	dirtyMark := ""
	if s.Dirty {
		dirtyMark = "*"
	}
	status := fmt.Sprintf(" %s%s [%d, %d]", s.FileName, dirtyMark, s.Line+1, s.Col+1)
	if len(status) > width {
		return status[:width]
	}
	for len(status) < width {
		status += " "
	}
	return status
}
