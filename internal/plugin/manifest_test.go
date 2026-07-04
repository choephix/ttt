package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	data := `{
		"name": "test-plugin",
		"description": "A test plugin",
		"version": "1.0.0",
		"author": "tester",
		"entry": "main.ttt.lua",
		"permissions": {
			"panel.sidebar": true,
			"commands": true,
			"system.exec": ["docker"]
		}
	}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", m.Name)
	}
	if m.Entry != "main.ttt.lua" {
		t.Errorf("expected entry main.ttt.lua, got %s", m.Entry)
	}
	if !m.Permissions.PanelSidebar {
		t.Error("expected panel.sidebar to be true")
	}
	if len(m.Permissions.SystemExec) != 1 || m.Permissions.SystemExec[0] != "docker" {
		t.Errorf("expected system.exec [docker], got %v", m.Permissions.SystemExec)
	}
}

func TestLoadManifestMissingName(t *testing.T) {
	dir := t.TempDir()
	data := `{"entry": "main.ttt.lua"}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadManifestMissingEntry(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "test"}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestLoadManifestMissingFile(t *testing.T) {
	_, err := LoadManifest(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadManifestAPIVersionDefaultsToOne(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "test", "entry": "init.lua"}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.API != 1 {
		t.Errorf("expected missing api to default to 1, got %d", m.API)
	}
}

func TestLoadManifestAPIVersionSupported(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "test", "entry": "init.lua", "api": 1}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.API != 1 {
		t.Errorf("expected api 1, got %d", m.API)
	}
}

func TestLoadManifestAPIVersionTooNew(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "test", "entry": "init.lua", "api": 2}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for unsupported api version")
	}
}

func TestLoadManifestAPIVersionInvalid(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "test", "entry": "init.lua", "api": -1}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for invalid api version")
	}
}

func TestLoadManifestNetworkHosts(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "t", "entry": "init.lua", "permissions": {"network.http": ["api.github.com"]}}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Permissions.NetworkHTTP.All {
		t.Error("array form must not be all-hosts")
	}
	if !m.Permissions.NetworkHTTP.AllowsHost("api.github.com") {
		t.Error("declared host should be allowed")
	}
}

func TestLoadManifestNetworkInvalid(t *testing.T) {
	dir := t.TempDir()
	data := `{"name": "t", "entry": "init.lua", "permissions": {"network.http": 42}}`
	os.WriteFile(filepath.Join(dir, "plugin.ttt.json"), []byte(data), 0644)

	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for non-bool/non-array network.http")
	}
}
