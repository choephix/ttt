package view

import "testing"

func TestStatusBarFields(t *testing.T) {
	sb := &StatusBar{FileName: "file.go", Line: 2, Col: 4, Dirty: true}
	if sb.FileName != "file.go" {
		t.Errorf("expected FileName 'file.go', got %q", sb.FileName)
	}
	if sb.Line != 2 {
		t.Errorf("expected Line 2, got %d", sb.Line)
	}
	if !sb.Dirty {
		t.Error("expected Dirty to be true")
	}
}
