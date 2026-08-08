package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/ttt/internal/config"
)

func TestLoadDirEntriesClassifiesSymlinkDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, entry := range LoadDirEntries(root, config.DefaultExplorerSettings()) {
		if entry.Name == "linked" {
			if !entry.IsDir {
				t.Fatal("symlinked directory is not expandable")
			}
			if entry.Path != link {
				t.Fatalf("path = %q, want %q", entry.Path, link)
			}
			return
		}
	}
	t.Fatal("symlink entry missing")
}

func TestLoadDirEntriesKeepsSymlinkFileAsFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, entry := range LoadDirEntries(root, config.DefaultExplorerSettings()) {
		if entry.Name == "linked.txt" {
			if entry.IsDir {
				t.Fatal("symlinked file was classified as a directory")
			}
			return
		}
	}
	t.Fatal("symlink entry missing")
}

func TestLoadDirEntriesFlagsSymlinks(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real.txt")
	if err := os.WriteFile(real, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "nowhere"), filepath.Join(root, "dangling.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	want := map[string]struct{ symlink, broken bool }{
		"real.txt":     {false, false},
		"linked.txt":   {true, false},
		"dangling.txt": {true, true},
	}
	seen := 0
	for _, entry := range LoadDirEntries(root, config.DefaultExplorerSettings()) {
		w, ok := want[entry.Name]
		if !ok {
			continue
		}
		seen++
		if entry.IsSymlink != w.symlink {
			t.Errorf("%s: IsSymlink = %v, want %v", entry.Name, entry.IsSymlink, w.symlink)
		}
		if entry.Broken != w.broken {
			t.Errorf("%s: Broken = %v, want %v", entry.Name, entry.Broken, w.broken)
		}
	}
	if seen != len(want) {
		t.Fatalf("found %d of %d entries", seen, len(want))
	}
}
