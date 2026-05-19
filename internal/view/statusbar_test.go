package view

import "testing"

func TestRenderStatusBar(t *testing.T) {
	sb := &StatusBar{FileName: "file.go", Line: 2, Col: 4, Dirty: true}
	s := sb.RenderStatusBar(30)
	if len(s) != 30 {
		t.Errorf("expected status bar length 30, got %d", len(s))
	}
	if s[:8] != " file.go" {
		t.Errorf("expected file name prefix with space, got %q", s[:8])
	}
	if s[8] != '*' {
		t.Errorf("expected dirty mark '*', got %q", s[8])
	}
}
