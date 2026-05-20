package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.AccentColor != "#4ec9b0" {
		t.Fatalf("expected AccentColor '#4ec9b0', got '%s'", th.AccentColor)
	}
	if th.ActiveTab.Bold != true {
		t.Fatal("expected ActiveTab.Bold true")
	}
}

func TestThemePartialJSON(t *testing.T) {
	th := DefaultTheme()
	json.Unmarshal([]byte(`{"statusBar": {"fg": "yellow", "bg": "#ff0000"}}`), &th)
	th.ResolveColors()

	if th.StatusBar.Fg != "yellow" {
		t.Fatalf("expected StatusBar.Fg 'yellow', got '%s'", th.StatusBar.Fg)
	}
	if th.StatusBar.Bg != "#ff0000" {
		t.Fatalf("expected StatusBar.Bg '#ff0000', got '%s'", th.StatusBar.Bg)
	}
	if th.ActiveTab.Fg != "#4ec9b0" {
		t.Fatalf("ActiveTab.Fg should still be '#4ec9b0', got '%s'", th.ActiveTab.Fg)
	}
}

func TestThemeHexColors(t *testing.T) {
	th := ThemeConfig{}
	json.Unmarshal([]byte(`{"lineNumber": {"fg": "#808080"}}`), &th)

	if th.LineNumber.Fg != "#808080" {
		t.Fatalf("expected '#808080', got '%s'", th.LineNumber.Fg)
	}
}
