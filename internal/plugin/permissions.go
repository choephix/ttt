package plugin

import (
	"fmt"
	"slices"
	"strings"
)

type PermissionSet struct {
	PanelSidebar bool     `json:"panel.sidebar,omitempty"`
	PanelBottom  bool     `json:"panel.bottom,omitempty"`
	PanelDrawer  bool     `json:"panel.drawer,omitempty"`
	PanelEditor  bool     `json:"panel.editor,omitempty"`
	Commands     bool     `json:"commands,omitempty"`
	Keybindings  bool     `json:"keybindings,omitempty"`
	EditorRead   bool     `json:"editor.read,omitempty"`
	EditorWrite  bool     `json:"editor.write,omitempty"`
	FsRead       bool     `json:"fs.read,omitempty"`
	FsWrite      bool     `json:"fs.write,omitempty"`
	SystemExec   []string `json:"system.exec,omitempty"`
	SystemEnv    bool     `json:"system.env,omitempty"`
	NetworkHTTP  bool     `json:"network.http,omitempty"`
	EventsFile   bool     `json:"events.file,omitempty"`
	EventsEditor bool     `json:"events.editor,omitempty"`
	Settings     bool     `json:"settings,omitempty"`
	SettingsKeys []string `json:"settings_keys,omitempty"`
}

type PermissionDiffEntry struct {
	Name  string
	Value string
}

type PermissionDiff struct {
	Entries []PermissionDiffEntry
}

func DiffPermissions(granted, requested PermissionSet) PermissionDiff {
	var entries []PermissionDiffEntry

	check := func(name string, g, r bool) {
		if r && !g {
			entries = append(entries, PermissionDiffEntry{Name: name, Value: "required"})
		}
	}

	check("panel.sidebar", granted.PanelSidebar, requested.PanelSidebar)
	check("panel.bottom", granted.PanelBottom, requested.PanelBottom)
	check("panel.drawer", granted.PanelDrawer, requested.PanelDrawer)
	check("panel.editor", granted.PanelEditor, requested.PanelEditor)
	check("commands", granted.Commands, requested.Commands)
	check("keybindings", granted.Keybindings, requested.Keybindings)
	check("editor.read", granted.EditorRead, requested.EditorRead)
	check("editor.write", granted.EditorWrite, requested.EditorWrite)
	check("fs.read", granted.FsRead, requested.FsRead)
	check("fs.write", granted.FsWrite, requested.FsWrite)
	check("system.env", granted.SystemEnv, requested.SystemEnv)
	check("network.http", granted.NetworkHTTP, requested.NetworkHTTP)
	check("events.file", granted.EventsFile, requested.EventsFile)
	check("events.editor", granted.EventsEditor, requested.EventsEditor)
	check("settings", granted.Settings, requested.Settings)

	grantedSettingsKeys := make(map[string]bool)
	for _, k := range granted.SettingsKeys {
		grantedSettingsKeys[k] = true
	}
	for _, k := range requested.SettingsKeys {
		if !grantedSettingsKeys[k] {
			entries = append(entries, PermissionDiffEntry{Name: "settings_keys", Value: k})
		}
	}

	grantedExec := make(map[string]bool)
	for _, b := range granted.SystemExec {
		grantedExec[b] = true
	}
	for _, b := range requested.SystemExec {
		if !grantedExec[b] {
			entries = append(entries, PermissionDiffEntry{Name: "system.exec", Value: b})
		}
	}

	return PermissionDiff{Entries: entries}
}

func (d PermissionDiff) IsEmpty() bool {
	return len(d.Entries) == 0
}

func (ps PermissionSet) Check(perm string) error {
	allowed := false
	switch perm {
	case "panel.sidebar":
		allowed = ps.PanelSidebar
	case "panel.bottom":
		allowed = ps.PanelBottom
	case "panel.drawer":
		allowed = ps.PanelDrawer
	case "panel.editor":
		allowed = ps.PanelEditor
	case "commands":
		allowed = ps.Commands
	case "keybindings":
		allowed = ps.Keybindings
	case "editor.read":
		allowed = ps.EditorRead
	case "editor.write":
		allowed = ps.EditorWrite
	case "fs.read":
		allowed = ps.FsRead
	case "fs.write":
		allowed = ps.FsWrite
	case "system.env":
		allowed = ps.SystemEnv
	case "network.http":
		allowed = ps.NetworkHTTP
	case "events.file":
		allowed = ps.EventsFile
	case "events.editor":
		allowed = ps.EventsEditor
	case "settings":
		allowed = ps.Settings
	default:
		return fmt.Errorf("unknown permission: %s", perm)
	}
	if !allowed {
		return fmt.Errorf("permission denied: %s", perm)
	}
	return nil
}

func (ps PermissionSet) CheckSettingsKey(key string) error {
	if !ps.Settings {
		return fmt.Errorf("permission denied: settings")
	}
	for _, pattern := range ps.SettingsKeys {
		if pattern == key {
			return nil
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(key, prefix) {
				return nil
			}
		}
	}
	return fmt.Errorf("permission denied: settings key %q", key)
}

func (ps PermissionSet) CheckExec(binary string) error {
	if slices.Contains(ps.SystemExec, binary) {
		return nil
	}
	return fmt.Errorf("permission denied: system.exec %q", binary)
}

func (ps PermissionSet) DisplayEntries() []PermissionDiffEntry {
	var entries []PermissionDiffEntry

	add := func(name string, val bool) {
		if val {
			entries = append(entries, PermissionDiffEntry{Name: name, Value: "yes"})
		}
	}

	add("Sidebar panel", ps.PanelSidebar)
	add("Bottom panel", ps.PanelBottom)
	add("Drawer", ps.PanelDrawer)
	add("Editor tab", ps.PanelEditor)
	add("Commands", ps.Commands)
	add("Keybindings", ps.Keybindings)
	add("Read editor", ps.EditorRead)
	add("Write editor", ps.EditorWrite)
	add("Read files", ps.FsRead)
	add("Write files", ps.FsWrite)
	add("Environment", ps.SystemEnv)
	add("HTTP requests", ps.NetworkHTTP)
	add("File events", ps.EventsFile)
	add("Editor events", ps.EventsEditor)

	for _, b := range ps.SystemExec {
		entries = append(entries, PermissionDiffEntry{Name: "Run binary", Value: b})
	}

	add("Settings", ps.Settings)
	for _, k := range ps.SettingsKeys {
		entries = append(entries, PermissionDiffEntry{Name: "Settings key", Value: k})
	}

	return entries
}
