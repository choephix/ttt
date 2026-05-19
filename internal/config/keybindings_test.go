package config

import (
	"testing"
)

func TestParseSingleKey(t *testing.T) {
	steps, err := ParseKeyString("ctrl+b")
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if !steps[0].Ctrl || steps[0].Rune != 'b' {
		t.Fatalf("expected Ctrl+b, got %+v", steps[0])
	}
}

func TestParseChord(t *testing.T) {
	steps, err := ParseKeyString("ctrl+k ctrl+c")
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if !steps[0].Ctrl || steps[0].Rune != 'k' {
		t.Fatalf("step 0: expected Ctrl+k, got %+v", steps[0])
	}
	if !steps[1].Ctrl || steps[1].Rune != 'c' {
		t.Fatalf("step 1: expected Ctrl+c, got %+v", steps[1])
	}
}

func TestParseSpecialKeys(t *testing.T) {
	tests := []struct {
		input   string
		keyName string
	}{
		{"escape", "Escape"},
		{"esc", "Escape"},
		{"enter", "Enter"},
		{"return", "Enter"},
		{"f5", "F5"},
		{"pgup", "PgUp"},
		{"pagedown", "PgDn"},
		{"space", "Space"},
		{"tab", "Tab"},
	}

	for _, tt := range tests {
		steps, err := ParseKeyString(tt.input)
		if err != nil {
			t.Fatalf("ParseKeyString(%q): %v", tt.input, err)
		}
		if steps[0].KeyName != tt.keyName {
			t.Fatalf("ParseKeyString(%q): expected KeyName %q, got %q", tt.input, tt.keyName, steps[0].KeyName)
		}
	}
}

func TestParseModifiers(t *testing.T) {
	steps, err := ParseKeyString("ctrl+shift+f5")
	if err != nil {
		t.Fatal(err)
	}
	if !steps[0].Ctrl || !steps[0].Shift {
		t.Fatalf("expected Ctrl+Shift, got %+v", steps[0])
	}
	if steps[0].KeyName != "F5" {
		t.Fatalf("expected F5, got %q", steps[0].KeyName)
	}
}

func TestParseAltModifier(t *testing.T) {
	steps, err := ParseKeyString("alt+x")
	if err != nil {
		t.Fatal(err)
	}
	if !steps[0].Alt || steps[0].Rune != 'x' {
		t.Fatalf("expected Alt+x, got %+v", steps[0])
	}
}

func TestParseCaseInsensitive(t *testing.T) {
	steps, err := ParseKeyString("Ctrl+B")
	if err != nil {
		t.Fatal(err)
	}
	if !steps[0].Ctrl || steps[0].Rune != 'b' {
		t.Fatalf("expected ctrl+b, got %+v", steps[0])
	}
}

func TestParseErrors(t *testing.T) {
	tests := []string{
		"",
		"ctrl+",
		"ctrl+abc",
		"foo+b",
	}

	for _, input := range tests {
		_, err := ParseKeyString(input)
		if err == nil {
			t.Fatalf("ParseKeyString(%q): expected error", input)
		}
	}
}

func TestDefaultKeybindingsParse(t *testing.T) {
	kb := DefaultKeybindings()
	err := ParseKeyBindings(kb)
	if err != nil {
		t.Fatalf("DefaultKeybindings should parse without error: %v", err)
	}
	for _, b := range kb {
		if len(b.Steps) == 0 {
			t.Fatalf("binding %q has no steps after parsing", b.Key)
		}
	}
}

func TestIsChord(t *testing.T) {
	kb := []KeyBinding{
		{Key: "ctrl+k ctrl+c", Command: "test.chord"},
		{Key: "ctrl+b", Command: "test.single"},
	}
	ParseKeyBindings(kb)

	if !kb[0].IsChord() {
		t.Fatal("ctrl+k ctrl+c should be a chord")
	}
	if kb[1].IsChord() {
		t.Fatal("ctrl+b should not be a chord")
	}
}

func TestParseShiftTab(t *testing.T) {
	steps, err := ParseKeyString("shift+tab")
	if err != nil {
		t.Fatal(err)
	}
	if !steps[0].Shift || steps[0].KeyName != "Tab" {
		t.Fatalf("expected Shift+Tab, got %+v", steps[0])
	}
}
