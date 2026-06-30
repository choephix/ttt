package plugin

import (
	"testing"
)

type mockSettingsAPI struct {
	store map[string]string
}

func (m *mockSettingsAPI) Get(key string) (string, bool) {
	v, ok := m.store[key]
	return v, ok
}

func (m *mockSettingsAPI) Set(key, value string) error {
	m.store[key] = value
	return nil
}

func setupTestPluginWithSettings(perms PermissionSet, api SettingsAPI) (*Plugin, func()) {
	p := &Plugin{
		Granted:  perms,
		Settings: api,
	}
	p.State = NewSandbox()
	setupSettingsModule(p.State, p)
	return p, func() { p.State.Close() }
}

func TestSettingsGet(t *testing.T) {
	api := &mockSettingsAPI{store: map[string]string{"formatters.go": "gofmt"}}
	p, cleanup := setupTestPluginWithSettings(
		PermissionSet{Settings: true, SettingsKeys: []string{"formatters.*"}},
		api,
	)
	defer cleanup()

	err := p.State.DoString(`
		local settings = require("ttt.settings")
		result = settings.get("formatters.go")
	`)
	if err != nil {
		t.Fatalf("DoString: %v", err)
	}
	val := p.State.GetGlobal("result").String()
	if val != "gofmt" {
		t.Errorf("expected 'gofmt', got %q", val)
	}
}

func TestSettingsGetNil(t *testing.T) {
	api := &mockSettingsAPI{store: map[string]string{}}
	p, cleanup := setupTestPluginWithSettings(
		PermissionSet{Settings: true, SettingsKeys: []string{"formatters.*"}},
		api,
	)
	defer cleanup()

	err := p.State.DoString(`
		local settings = require("ttt.settings")
		result = settings.get("formatters.py")
	`)
	if err != nil {
		t.Fatalf("DoString: %v", err)
	}
	val := p.State.GetGlobal("result").String()
	if val != "nil" {
		t.Errorf("expected 'nil', got %q", val)
	}
}

func TestSettingsSet(t *testing.T) {
	api := &mockSettingsAPI{store: map[string]string{}}
	p, cleanup := setupTestPluginWithSettings(
		PermissionSet{Settings: true, SettingsKeys: []string{"formatters.*"}},
		api,
	)
	defer cleanup()

	err := p.State.DoString(`
		local settings = require("ttt.settings")
		settings.set("formatters.js", "prettier --stdin-filepath {file}")
	`)
	if err != nil {
		t.Fatalf("DoString: %v", err)
	}
	if api.store["formatters.js"] != "prettier --stdin-filepath {file}" {
		t.Errorf("expected prettier command, got %q", api.store["formatters.js"])
	}
}

func TestSettingsPermissionDenied(t *testing.T) {
	api := &mockSettingsAPI{store: map[string]string{}}
	p, cleanup := setupTestPluginWithSettings(
		PermissionSet{Settings: true, SettingsKeys: []string{"formatters.*"}},
		api,
	)
	defer cleanup()

	err := p.State.DoString(`
		local settings = require("ttt.settings")
		settings.get("editor.tabSize")
	`)
	if err == nil {
		t.Fatal("expected permission error")
	}
}

func TestSettingsNoPermission(t *testing.T) {
	api := &mockSettingsAPI{store: map[string]string{}}
	p, cleanup := setupTestPluginWithSettings(
		PermissionSet{},
		api,
	)
	defer cleanup()

	err := p.State.DoString(`
		local settings = require("ttt.settings")
		settings.get("formatters.go")
	`)
	if err == nil {
		t.Fatal("expected permission error")
	}
}

func TestSettingsExactKeyMatch(t *testing.T) {
	api := &mockSettingsAPI{store: map[string]string{"formatters.go": "gofmt"}}
	p, cleanup := setupTestPluginWithSettings(
		PermissionSet{Settings: true, SettingsKeys: []string{"formatters.go"}},
		api,
	)
	defer cleanup()

	err := p.State.DoString(`
		local settings = require("ttt.settings")
		result = settings.get("formatters.go")
	`)
	if err != nil {
		t.Fatalf("DoString: %v", err)
	}
	val := p.State.GetGlobal("result").String()
	if val != "gofmt" {
		t.Errorf("expected 'gofmt', got %q", val)
	}

	err = p.State.DoString(`
		local settings = require("ttt.settings")
		settings.get("formatters.py")
	`)
	if err == nil {
		t.Fatal("expected permission error for non-matching exact key")
	}
}

func TestCheckSettingsKey(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		key     string
		wantErr bool
	}{
		{"wildcard match", []string{"formatters.*"}, "formatters.go", false},
		{"wildcard no match", []string{"formatters.*"}, "editor.tabSize", true},
		{"exact match", []string{"formatters.go"}, "formatters.go", false},
		{"exact no match", []string{"formatters.go"}, "formatters.py", true},
		{"no settings permission", nil, "formatters.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := PermissionSet{Settings: len(tt.keys) > 0, SettingsKeys: tt.keys}
			err := ps.CheckSettingsKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSettingsKey(%q) error = %v, wantErr = %v", tt.key, err, tt.wantErr)
			}
		})
	}
}
