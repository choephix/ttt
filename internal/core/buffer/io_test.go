package buffer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoadAndSaveFile(t *testing.T) {
	fname := "testfile.txt"
	defer os.Remove(fname)
	orig := &Buffer{Lines: []string{"hello", "world"}}
	if err := orig.SaveFile(fname); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	b := &Buffer{}
	if err := b.LoadFile(fname); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if len(b.Lines) != 2 || b.Lines[0] != "hello" || b.Lines[1] != "world" {
		t.Errorf("expected [hello world], got %v", b.Lines)
	}
}

func TestSavePreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits not meaningful on Windows")
	}
	dir := t.TempDir()
	fname := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(fname, []byte("old\n"), 0755); err != nil {
		t.Fatal(err)
	}
	b := &Buffer{Lines: []string{"new", ""}}
	if err := b.SaveFile(fname); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	info, err := os.Stat(fname)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected mode 0755 preserved, got %o", info.Mode().Perm())
	}
}

func TestSaveThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on Windows")
	}
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(realFile, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, link); err != nil {
		t.Fatal(err)
	}
	b := &Buffer{Lines: []string{"new", ""}}
	if err := b.SaveFile(link); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	// The link must still be a symlink, and the real file must hold new content.
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("symlink was replaced with a regular file")
	}
	data, err := os.ReadFile(realFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new\n" {
		t.Errorf("expected real file to contain %q, got %q", "new\n", string(data))
	}
}

func TestDiskChanged(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fname, []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &Buffer{}
	if err := b.LoadFile(fname); err != nil {
		t.Fatal(err)
	}
	if b.DiskChanged(fname) {
		t.Errorf("expected no change right after load")
	}

	// Simulate an external edit: different size and a later mtime.
	if err := os.WriteFile(fname, []byte("changed externally\n"), 0644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(fname, future, future); err != nil {
		t.Fatal(err)
	}
	if !b.DiskChanged(fname) {
		t.Errorf("expected change to be detected after external edit")
	}

	// Saving re-syncs our recorded disk state.
	if err := b.SaveFile(fname); err != nil {
		t.Fatal(err)
	}
	if b.DiskChanged(fname) {
		t.Errorf("expected no change right after save")
	}
}

func TestDiskChangedNewBuffer(t *testing.T) {
	b := &Buffer{Lines: []string{"never saved"}}
	if b.DiskChanged("/nonexistent/path") {
		t.Errorf("a buffer with no recorded disk state should report no change")
	}
}

func TestLoadDetectsLF(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "lf.txt")
	os.WriteFile(fname, []byte("line1\nline2\nline3\n"), 0644)
	b := &Buffer{}
	if err := b.LoadFile(fname); err != nil {
		t.Fatal(err)
	}
	if b.LineEnding != "\n" {
		t.Errorf("expected LF, got %q", b.LineEnding)
	}
}

func TestLoadDetectsCRLF(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "crlf.txt")
	os.WriteFile(fname, []byte("line1\r\nline2\r\nline3\r\n"), 0644)
	b := &Buffer{}
	if err := b.LoadFile(fname); err != nil {
		t.Fatal(err)
	}
	if b.LineEnding != "\r\n" {
		t.Errorf("expected CRLF, got %q", b.LineEnding)
	}
}

func TestSavePreservesLF(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "lf.txt")
	os.WriteFile(fname, []byte("a\nb"), 0644)
	b := &Buffer{}
	b.LoadFile(fname)
	b.Lines[0] = "edited"
	b.SaveFile(fname)
	data, _ := os.ReadFile(fname)
	if string(data) != "edited\nb" {
		t.Errorf("expected LF preserved, got %q", string(data))
	}
}

func TestSavePreservesCRLF(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "crlf.txt")
	os.WriteFile(fname, []byte("a\r\nb"), 0644)
	b := &Buffer{}
	b.LoadFile(fname)
	b.Lines[0] = "edited"
	b.SaveFile(fname)
	data, _ := os.ReadFile(fname)
	if string(data) != "edited\r\nb" {
		t.Errorf("expected CRLF preserved, got %q", string(data))
	}
}

func TestNewBufferDefaultsToLF(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "new.txt")
	b := &Buffer{Lines: []string{"hello", "world"}}
	b.SaveFile(fname)
	data, _ := os.ReadFile(fname)
	if string(data) != "hello\nworld" {
		t.Errorf("expected LF default, got %q", string(data))
	}
}

func TestSaveTrimsTrailingWhitespace(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "trim.txt")
	b := &Buffer{
		Lines:                  []string{"hello   ", "world\t\t", "clean"},
		TrimTrailingWhitespace: true,
	}
	b.SaveFile(fname)
	data, _ := os.ReadFile(fname)
	if string(data) != "hello\nworld\nclean" {
		t.Errorf("expected trimmed output, got %q", string(data))
	}
}

func TestSaveWithoutTrimPreservesWhitespace(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "notrim.txt")
	b := &Buffer{
		Lines: []string{"hello   ", "world\t\t"},
	}
	b.SaveFile(fname)
	data, _ := os.ReadFile(fname)
	if string(data) != "hello   \nworld\t\t" {
		t.Errorf("expected whitespace preserved, got %q", string(data))
	}
}

func TestSaveDoesNotLeaveTempFiles(t *testing.T) {
	dir := t.TempDir()
	fname := filepath.Join(dir, "file.txt")
	b := &Buffer{Lines: []string{"content", ""}}
	if err := b.SaveFile(fname); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "file.txt" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected only file.txt, got %v", names)
	}
}
