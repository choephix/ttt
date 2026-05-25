package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditorGroupOpenFileNotFound(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false)
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.OpenFile("/nonexistent/path/file.txt")

	if errMsg == "" {
		t.Fatal("expected OnError to be called for missing file")
	}
	if len(g.tabs) != 1 || g.tabs[0].FilePath != "untitled" {
		t.Fatal("should not have added a tab for missing file")
	}
}

func TestEditorGroupOpenFileSuccess(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello\nworld"), 0644)

	g := NewEditorGroupWidget(nil, 4, false)
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.OpenFile(path)

	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if g.tabs[g.active].FilePath != path {
		t.Fatalf("expected active tab to be %s, got %s", path, g.tabs[g.active].FilePath)
	}
}

func TestEditorGroupSaveError(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false)
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.tabs[0].FilePath = "/nonexistent/dir/file.txt"
	g.tabs[0].Buf.Lines = []string{"test"}

	ok := g.Save()
	if ok {
		t.Fatal("Save should return false on error")
	}
	if errMsg == "" {
		t.Fatal("expected OnError to be called for save failure")
	}
}

func TestEditorGroupSaveSuccess(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.txt")

	g := NewEditorGroupWidget(nil, 4, false)
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.tabs[0].FilePath = path
	g.tabs[0].Buf.Lines = []string{"saved content"}

	ok := g.Save()
	if !ok {
		t.Fatal("Save should return true on success")
	}
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "saved content" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestEditorGroupSaveAsError(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false)
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.SaveAs("/nonexistent/dir/file.txt")

	if errMsg == "" {
		t.Fatal("expected OnError to be called for save-as failure")
	}
	if g.tabs[0].FilePath != "untitled" {
		t.Fatal("FilePath should not change on failed SaveAs")
	}
}

func TestEditorGroupSaveAsSuccess(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "saved.go")

	g := NewEditorGroupWidget(nil, 4, false)
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.tabs[0].Buf.Lines = []string{"package main"}
	g.SaveAs(path)

	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if g.tabs[0].FilePath != path {
		t.Fatalf("expected FilePath to be %s, got %s", path, g.tabs[0].FilePath)
	}
}
