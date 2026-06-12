package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.Editor.TabSize != 4 {
		t.Fatalf("expected TabSize 4, got %d", s.Editor.TabSize)
	}
	if !s.Editor.InsertSpaces {
		t.Fatal("expected InsertSpaces true")
	}
	if !s.SidebarVisible {
		t.Fatal("expected SidebarVisible true")
	}
	if s.SidebarWidth != 30 {
		t.Fatalf("expected SidebarWidth 30, got %d", s.SidebarWidth)
	}
}

func TestSettingsPartialJSON(t *testing.T) {
	s := DefaultSettings()
	json.Unmarshal([]byte(`{"editor": {"tabSize": 2}}`), &s)

	if s.Editor.TabSize != 2 {
		t.Fatalf("expected TabSize 2, got %d", s.Editor.TabSize)
	}
	if !s.Editor.InsertSpaces {
		t.Fatal("InsertSpaces should still be true (not in JSON)")
	}
	if s.SidebarWidth != 30 {
		t.Fatalf("SidebarWidth should still be 30, got %d", s.SidebarWidth)
	}
}

func TestSettingsEmptyJSON(t *testing.T) {
	s := DefaultSettings()
	json.Unmarshal([]byte(`{}`), &s)

	if s.Editor.TabSize != 4 {
		t.Fatalf("expected TabSize 4, got %d", s.Editor.TabSize)
	}
}
