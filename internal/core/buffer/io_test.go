package buffer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
