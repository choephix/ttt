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
}

func TestSettingsEmptyJSON(t *testing.T) {
	s := DefaultSettings()
	json.Unmarshal([]byte(`{}`), &s)

	if s.Editor.TabSize != 4 {
		t.Fatalf("expected TabSize 4, got %d", s.Editor.TabSize)
	}
}
