package buffer

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
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

// SaveFile writes the buffer contents to a file atomically: it writes to a
// temporary file in the same directory, fsyncs it, then renames it over the
// target. This guarantees the file on disk is always either the complete old
// or complete new content, never a truncated partial write. The target's
// permissions are preserved; symlinks are followed so the link is kept intact.
func (b *Buffer) SaveFile(filename string) error {
	if b.InsertFinalNewline && (len(b.Lines) == 0 || b.Lines[len(b.Lines)-1] != "") {
		b.Lines = append(b.Lines, "")
	}

	// Resolve symlinks so we write through to the real file rather than
	// replacing the link itself with a regular file on rename.
	target := filename
	if resolved, err := filepath.EvalSymlinks(filename); err == nil {
		target = resolved
	}

	dir := filepath.Dir(target)
	mode := os.FileMode(0644)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, ".ttt-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail out before the rename succeeds.
	defer os.Remove(tmpName)

	w := bufio.NewWriter(tmp)
	if err := b.writeLines(w); err != nil {
		tmp.Close()
		return err
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return err
	}

	b.Dirty = false
	return nil
}

func (b *Buffer) writeLines(w io.Writer) error {
	for i, line := range b.Lines {
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
		if i < len(b.Lines)-1 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}
