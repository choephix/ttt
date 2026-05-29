package buffer

import (
	"bufio"
	"os"
)

// LoadFile loads a file into the buffer, replacing its contents.
func (b *Buffer) LoadFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	if b.InsertFinalNewline && (len(lines) == 0 || lines[len(lines)-1] != "") {
		lines = append(lines, "")
	}
	b.Lines = lines
	b.Dirty = false
	return nil
}

// SaveFile writes the buffer contents to a file.
func (b *Buffer) SaveFile(filename string) error {
	if b.InsertFinalNewline && (len(b.Lines) == 0 || b.Lines[len(b.Lines)-1] != "") {
		b.Lines = append(b.Lines, "")
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for i, line := range b.Lines {
		if _, err := w.WriteString(line); err != nil {
			return err
		}
		if i < len(b.Lines)-1 {
			if err := w.WriteByte('\n'); err != nil {
				return err
			}
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	b.Dirty = false
	return nil
}
