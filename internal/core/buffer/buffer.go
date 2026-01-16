package buffer

// Buffer represents a text buffer with line-based storage.
type Buffer struct {
	Lines []string
	Dirty bool
}

// InsertRune inserts a rune at the given line and column.
func (b *Buffer) InsertRune(line, col int, r rune) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	l := b.Lines[line]
	if col < 0 || col > len([]rune(l)) {
		return
	}
	runes := []rune(l)
	runes = append(runes[:col], append([]rune{r}, runes[col:]...)...)
	b.Lines[line] = string(runes)
	b.Dirty = true
}

// DeleteRune deletes a rune at the given line and column.
func (b *Buffer) DeleteRune(line, col int) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	l := b.Lines[line]
	runes := []rune(l)
	if col < 0 || col >= len(runes) {
		return
	}
	runes = append(runes[:col], runes[col+1:]...)
	b.Lines[line] = string(runes)
	b.Dirty = true
}

// InsertLine inserts a new line at the given index.
func (b *Buffer) InsertLine(idx int, text string) {
	if idx < 0 || idx > len(b.Lines) {
		return
	}
	b.Lines = append(b.Lines[:idx], append([]string{text}, b.Lines[idx:]...)...)
	b.Dirty = true
}

// DeleteLine deletes the line at the given index.
func (b *Buffer) DeleteLine(idx int) {
	if idx < 0 || idx >= len(b.Lines) {
		return
	}
	b.Lines = append(b.Lines[:idx], b.Lines[idx+1:]...)
	b.Dirty = true
}
