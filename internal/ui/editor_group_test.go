package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/ttt/internal/term"
)

func TestEditorGroupShebangHighlighting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deploy")
	text := "#!/usr/bin/env python3\nprint(\"deploy\")\n"
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"open", "readonly", "buffer", "save-as", "rename", "refresh", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			g := NewEditorGroupWidget(nil, 4, false, "extended")
			g.SyntaxHighlight = mode != "disabled"
			switch mode {
			case "readonly":
				g.OpenFileReadOnly(path, "deploy")
			case "buffer":
				g.OpenBufferReadOnly("deploy", path, []string{"#!/usr/bin/env python3", `print("deploy")`})
			default:
				g.OpenFile(path)
			}
			switch mode {
			case "save-as":
				if !g.SaveAs(filepath.Join(t.TempDir(), "renamed")) {
					t.Fatal("SaveAs failed")
				}
			case "rename":
				g.RenamePath(path, filepath.Join(t.TempDir(), "renamed"))
			case "refresh":
				g.RefreshHighlighters()
			}
			h := g.Editor.Highlighter
			if mode == "disabled" {
				if h != nil {
					t.Fatal("highlighting should remain disabled")
				}
				return
			}
			if h == nil || h.Language() != "Python" {
				t.Fatalf("highlighter = %v, want Python", h)
			}
			for _, span := range h.HighlightLineAt(g.Editor.Buf.Lines, 1) {
				if span.Style == term.StyleSyntaxString {
					return
				}
			}
			t.Fatal("script string is not highlighted")
		})
	}
}

func TestOpenPluginTabReplacementClosesPreviousContent(t *testing.T) {
	group := NewEditorGroupWidget(nil, 4, true, "relative")
	closed := false
	notified := ""
	group.OnContentTabClose = func(id string) { notified = id }
	first := NewCommitDetailWidget("/repo", "ref", "abcdef0", false)
	first.OnClose = func() { closed = true }
	group.OpenPluginTab("commit", "Commit", first)
	group.OpenPluginTab("commit", "Replacement", NewCommitDetailWidget("/repo", "other", "1234567", false))
	if !closed {
		t.Fatal("replacing a content tab did not close its previous incarnation")
	}
	if notified != "commit" {
		t.Fatalf("replacement close notification = %q, want commit", notified)
	}
}

func TestInitialTabIsVirtual(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")
	if !g.tabs[0].Virtual {
		t.Fatal("expected initial tab to be virtual")
	}
}

func TestNewFileAlwaysCreatesTab(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")
	if len(g.tabs) != 1 || g.tabs[0].FilePath != "untitled" {
		t.Fatal("expected initial untitled tab")
	}

	g.tabs[0].Buf.Lines = []string{"some content"}
	g.tabs[0].Buf.Dirty = true

	g.NewFile()
	if len(g.tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(g.tabs))
	}
	if g.tabs[1].FilePath != "untitled-2" {
		t.Errorf("expected 'untitled-2', got %q", g.tabs[1].FilePath)
	}
	if g.active != 1 {
		t.Errorf("expected active tab 1, got %d", g.active)
	}
}

func TestNewFileSequentialNaming(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")

	g.NewFile()
	g.NewFile()
	g.NewFile()

	if len(g.tabs) != 4 {
		t.Fatalf("expected 4 tabs, got %d", len(g.tabs))
	}
	expected := []string{"untitled", "untitled-2", "untitled-3", "untitled-4"}
	for i, want := range expected {
		if g.tabs[i].FilePath != want {
			t.Errorf("tab %d: got %q, want %q", i, g.tabs[i].FilePath, want)
		}
	}
}

func TestNewFileReusesNameAfterClose(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.NewFile() // untitled-2

	g.SwitchTab(0)
	g.CloseTab()

	if g.tabs[0].FilePath != "untitled-2" {
		t.Fatalf("expected remaining tab 'untitled-2', got %q", g.tabs[0].FilePath)
	}

	g.NewFile()
	names := make(map[string]bool)
	for _, tab := range g.tabs {
		names[tab.FilePath] = true
	}
	if !names["untitled"] {
		t.Error("expected 'untitled' to be reused after close")
	}
}

func TestEditorGroupOpenFileNotFound(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	path := "/nonexistent/path/file.txt"
	g.OpenFile(path)

	if errMsg != "" {
		t.Fatalf("unexpected error for new file: %s", errMsg)
	}
	if g.tabs[g.active].FilePath != path {
		t.Fatalf("expected tab with path %q, got %q", path, g.tabs[g.active].FilePath)
	}
	if g.tabs[g.active].Buf.Lines[0] != "" {
		t.Fatal("expected empty buffer for new file")
	}
}

func TestEditorGroupOpenFileSuccess(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello\nworld"), 0644)

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.OpenFile(path)

	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if len(g.tabs) != 1 {
		t.Fatalf("expected 1 tab replacing untitled, got %d", len(g.tabs))
	}
	if g.tabs[g.active].FilePath != path {
		t.Fatalf("expected active tab to be %s, got %s", path, g.tabs[g.active].FilePath)
	}
}

func TestEditorGroupPreviewReusesUncommittedTab(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(first, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.PreviewFile(first)
	g.PreviewFile(second)

	if len(g.tabs) != 1 {
		t.Fatalf("preview should reuse one tab, got %d", len(g.tabs))
	}
	if g.tabs[0].FilePath != second || !g.tabs[0].Preview {
		t.Fatalf("active preview = %#v, want uncommitted %q", g.tabs[0], second)
	}
}

func TestEditorGroupCommittedOpenPreservesTab(t *testing.T) {
	dir := t.TempDir()
	preview := filepath.Join(dir, "preview.txt")
	next := filepath.Join(dir, "next.txt")
	for _, path := range []string{preview, next} {
		if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.PreviewFile(preview)
	g.OpenFile(preview)
	g.PreviewFile(next)

	if len(g.tabs) != 2 {
		t.Fatalf("preview after committed tab should create a second tab, got %d", len(g.tabs))
	}
	if g.tabs[0].Preview || g.tabs[0].FilePath != preview {
		t.Fatal("committed file was not preserved")
	}
	if !g.tabs[1].Preview || g.tabs[1].FilePath != next {
		t.Fatal("new preview should remain uncommitted")
	}
}

func TestEditorGroupDoubleClickCommitsPreviewTab(t *testing.T) {
	dir := t.TempDir()
	preview := filepath.Join(dir, "preview.txt")
	next := filepath.Join(dir, "next.txt")
	for _, path := range []string{preview, next} {
		if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.PreviewFile(preview)
	g.TabBar.OnTabDoubleClick(0)

	if g.tabs[0].Preview || g.TabBar.Tabs[0].Preview {
		t.Fatal("double-clicked preview should become a committed tab")
	}
	g.PreviewFile(next)
	if len(g.tabs) != 2 || g.tabs[0].FilePath != preview {
		t.Fatal("next preview replaced the double-clicked tab")
	}
}

func TestEditorGroupDirtyPreviewBecomesCommitted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preview.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.PreviewFile(path)
	g.tabs[g.active].Buf.Dirty = true
	g.syncTabs()

	if g.tabs[g.active].Preview {
		t.Fatal("editing a preview should commit it")
	}
	if g.TabBar.Tabs[g.active].Preview {
		t.Fatal("edited tab should no longer render as a preview")
	}
}

func TestEditorGroupCloseBackgroundTabKeepsActiveTab(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.NewFile()
	g.NewFile()
	g.SwitchTab(2)
	activePath := g.tabs[g.active].FilePath

	g.CloseTabAt(0)

	if len(g.tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(g.tabs))
	}
	if g.tabs[g.active].FilePath != activePath {
		t.Fatalf("closing a background tab activated %q, want %q", g.tabs[g.active].FilePath, activePath)
	}
}

func TestEditorGroupOpenFileKeepsTouchedUntitled(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.tabs[0].Buf.Lines = []string{"typed text"}
	g.tabs[0].Buf.Dirty = true

	g.OpenFile(path)

	if len(g.tabs) != 2 {
		t.Fatalf("expected 2 tabs when untitled is dirty, got %d", len(g.tabs))
	}
	if g.tabs[0].FilePath != "untitled" || g.tabs[1].FilePath != path {
		t.Fatalf("expected [untitled, %s], got [%s, %s]", path, g.tabs[0].FilePath, g.tabs[1].FilePath)
	}
}

func TestEditorGroupOpenFileKeepsNewFileTab(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.NewFile()
	g.SwitchTab(0)
	g.CloseTab()

	g.OpenFile(path)

	if len(g.tabs) != 2 {
		t.Fatalf("expected 2 tabs when user created new file, got %d", len(g.tabs))
	}
	if g.tabs[0].FilePath != "untitled-2" || g.tabs[1].FilePath != path {
		t.Fatalf("expected [untitled-2, %s], got [%s, %s]", path, g.tabs[0].FilePath, g.tabs[1].FilePath)
	}
}

func TestEditorGroupOpenFileKeepsPinnedPlaceholder(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	g.TogglePinTab()

	g.OpenFile(path)

	if len(g.tabs) != 2 {
		t.Fatalf("expected 2 tabs when placeholder is pinned, got %d", len(g.tabs))
	}
	if g.tabs[0].FilePath != "untitled" || g.tabs[1].FilePath != path {
		t.Fatalf("expected [untitled, %s], got [%s, %s]", path, g.tabs[0].FilePath, g.tabs[1].FilePath)
	}
	if g.pinnedCount != 1 {
		t.Fatalf("expected pinnedCount to remain 1, got %d", g.pinnedCount)
	}
	if g.IsActiveTabPinned() {
		t.Fatal("expected newly opened tab to not be pinned")
	}
}

func TestEditorGroupSaveError(t *testing.T) {
	g := NewEditorGroupWidget(nil, 4, false, "extended")
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.tabs[0].FilePath = "/nonexistent/dir/file.txt"
	g.tabs[0].Virtual = false
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

	g := NewEditorGroupWidget(nil, 4, false, "extended")
	var errMsg string
	g.OnError = func(msg string) { errMsg = msg }

	g.tabs[0].FilePath = path
	g.tabs[0].Virtual = false
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
	g := NewEditorGroupWidget(nil, 4, false, "extended")
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

	g := NewEditorGroupWidget(nil, 4, false, "extended")
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
	if g.tabs[0].Virtual {
		t.Fatal("SaveAs should clear Virtual flag")
	}
}
