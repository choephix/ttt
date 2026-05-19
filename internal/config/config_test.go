package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNoFiles(t *testing.T) {
	cfg := Load()
	if len(cfg.Keybindings) != 20 {
		t.Fatalf("expected 20 default keybindings, got %d", len(cfg.Keybindings))
	}
	if cfg.Settings.TabSize != 4 {
		t.Fatalf("expected default TabSize 4, got %d", cfg.Settings.TabSize)
	}
	if cfg.Theme.AccentColor != "darkcyan" {
		t.Fatalf("expected default AccentColor 'darkcyan', got '%s'", cfg.Theme.AccentColor)
	}
}

func TestReadFirst(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.json"), []byte(`{"ok":true}`), 0644)

	data, err := readFirst([]string{"/nonexistent", dir}, "test.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("unexpected data: %s", data)
	}
}

func TestReadFirstPriority(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "test.json"), []byte(`"first"`), 0644)
	os.WriteFile(filepath.Join(dir2, "test.json"), []byte(`"second"`), 0644)

	data, err := readFirst([]string{dir1, dir2}, "test.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"first"` {
		t.Fatalf("expected first dir to win, got: %s", data)
	}
}

func TestReadFirstMissing(t *testing.T) {
	_, err := readFirst([]string{"/nonexistent"}, "test.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
