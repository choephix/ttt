package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.StatusBar.Fg != "white" {
		t.Fatalf("expected StatusBar.Fg 'white', got '%s'", th.StatusBar.Fg)
	}
	if th.ActiveTab.Bold != true {
		t.Fatal("expected ActiveTab.Bold true")
	}
}

func TestThemePartialJSON(t *testing.T) {
	th := DefaultTheme()
	json.Unmarshal([]byte(`{"statusBar": {"fg": "yellow", "bg": "#ff0000"}}`), &th)

	if th.StatusBar.Fg != "yellow" {
		t.Fatalf("expected StatusBar.Fg 'yellow', got '%s'", th.StatusBar.Fg)
	}
	if th.StatusBar.Bg != "#ff0000" {
		t.Fatalf("expected StatusBar.Bg '#ff0000', got '%s'", th.StatusBar.Bg)
	}
	// Other styles unchanged
	if th.ActiveTab.Fg != "white" {
		t.Fatalf("ActiveTab.Fg should still be 'white', got '%s'", th.ActiveTab.Fg)
	}
}

func TestThemeHexColors(t *testing.T) {
	th := ThemeConfig{}
	json.Unmarshal([]byte(`{"lineNumber": {"fg": "#808080"}}`), &th)

	if th.LineNumber.Fg != "#808080" {
		t.Fatalf("expected '#808080', got '%s'", th.LineNumber.Fg)
	}
}
