package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Manifest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Version     string        `json:"version"`
	Author      string        `json:"author"`
	Entry       string        `json:"entry"`
	Permissions PermissionSet `json:"permissions"`
}

func LoadManifest(dir string) (Manifest, error) {
	path := filepath.Join(dir, "plugin.ttt.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	if m.Name == "" {
		return Manifest{}, fmt.Errorf("manifest missing required field: name")
	}
	if m.Entry == "" {
		return Manifest{}, fmt.Errorf("manifest missing required field: entry")
	}

	return m, nil
}
