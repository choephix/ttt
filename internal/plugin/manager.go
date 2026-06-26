package plugin

import (
	"log/slog"
	"os"

	"github.com/eugenioenko/ttt/internal/config"
)

type SidebarRegistration struct {
	ID     string
	Title  string
	Widget *PluginPanelWidget
}

type BottomRegistration struct {
	ID     string
	Title  string
	Widget *PluginPanelWidget
}

type Manager struct {
	plugins    []*Plugin
	registry   *Registry
	pluginsDir string

	SidebarPanels []SidebarRegistration
	BottomPanels  []BottomRegistration
}

func NewManager(pluginsDir string) *Manager {
	return &Manager{pluginsDir: pluginsDir}
}

func (m *Manager) LoadAll() []*Plugin {
	os.MkdirAll(m.pluginsDir, 0755)

	regPath := config.ConfigFilePath("plugins.ttt.json")
	reg, err := LoadRegistry(regPath)
	if err != nil {
		slog.Error("load plugin registry", "error", err)
		reg = &Registry{path: regPath}
	}
	m.registry = reg

	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		slog.Error("read plugins directory", "error", err)
		return nil
	}

	var needsApproval []*Plugin

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := m.pluginsDir + "/" + entry.Name()
		manifest, err := LoadManifest(dir)
		if err != nil {
			slog.Warn("skip plugin", "dir", entry.Name(), "error", err)
			continue
		}

		p := &Plugin{
			Name:     manifest.Name,
			Dir:      dir,
			Manifest: manifest,
		}

		regEntry := m.registry.Find(manifest.Name)
		if regEntry == nil {
			needsApproval = append(needsApproval, p)
			continue
		}

		if !regEntry.Enabled {
			continue
		}

		diff := DiffPermissions(regEntry.Permissions, manifest.Permissions)
		if !diff.IsEmpty() {
			regEntry.Enabled = false
			needsApproval = append(needsApproval, p)
			continue
		}

		p.Granted = regEntry.Permissions
		if err := p.Init(); err != nil {
			continue
		}

		m.plugins = append(m.plugins, p)
		m.collectRegistrations(p)
	}

	return needsApproval
}

func (m *Manager) collectRegistrations(p *Plugin) {
	if p.SidebarTitle != "" && p.RenderFunc != nil {
		m.SidebarPanels = append(m.SidebarPanels, SidebarRegistration{
			ID:     "plugin." + p.Name,
			Title:  p.SidebarTitle,
			Widget: NewPluginPanelWidget(p, p.RenderFunc, p.EventFunc),
		})
	}
	if p.BottomTitle != "" && p.BottomRenderFunc != nil {
		m.BottomPanels = append(m.BottomPanels, BottomRegistration{
			ID:     "plugin." + p.Name,
			Title:  p.BottomTitle,
			Widget: NewPluginPanelWidget(p, p.BottomRenderFunc, p.BottomEventFunc),
		})
	}
}

func (m *Manager) ApproveAndLoad(p *Plugin) error {
	p.Granted = p.Manifest.Permissions

	m.registry.AddOrUpdate(
		p.Manifest.Name,
		"",
		p.Manifest.Version,
		p.Manifest.Permissions,
	)
	if err := m.registry.Save(); err != nil {
		slog.Error("save plugin registry", "error", err)
	}

	if err := p.Init(); err != nil {
		return err
	}

	m.plugins = append(m.plugins, p)
	m.collectRegistrations(p)
	return nil
}

func (m *Manager) Shutdown() {
	for _, p := range m.plugins {
		p.Destroy()
	}
}

func (m *Manager) Plugins() []*Plugin {
	return m.plugins
}

func (m *Manager) PluginCount() int {
	return len(m.plugins)
}
