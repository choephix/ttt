package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
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

type GitRunner interface {
	Clone(url, targetDir string) error
	Pull(dir string) error
}

type execGitRunner struct{}

func (execGitRunner) Clone(url, targetDir string) error {
	cmd := exec.Command("git", "clone", url, targetDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (execGitRunner) Pull(dir string) error {
	cmd := exec.Command("git", "-C", dir, "pull")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type Manager struct {
	plugins      []*Plugin
	registry     *Registry
	pluginsDir   string
	registryPath string
	extraDirs    []string
	git          GitRunner

	SidebarPanels []SidebarRegistration
	BottomPanels  []BottomRegistration
}

func NewManager(pluginsDir, registryPath string, extraDirs ...string) *Manager {
	return &Manager{pluginsDir: pluginsDir, registryPath: registryPath, git: execGitRunner{}, extraDirs: extraDirs}
}

func (m *Manager) LoadAll() []*Plugin {
	os.MkdirAll(m.pluginsDir, 0755)

	regPath := m.registryPath
	reg, err := LoadRegistry(regPath)
	if err != nil {
		slog.Error("load plugin registry", "error", err)
		reg = &Registry{path: regPath}
	}
	m.registry = reg

	var needsApproval []*Plugin
	seen := map[string]bool{}

	dirs := append([]string{m.pluginsDir}, m.extraDirs...)
	for _, pluginDir := range dirs {
		entries, err := os.ReadDir(pluginDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			dir := filepath.Join(pluginDir, entry.Name())
			manifest, err := LoadManifest(dir)
			if err != nil {
				slog.Warn("skip plugin", "dir", entry.Name(), "error", err)
				continue
			}

			if seen[manifest.Name] {
				continue
			}
			seen[manifest.Name] = true

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
	}

	return needsApproval
}

func (m *Manager) collectRegistrations(p *Plugin) {
	id := "plugin." + p.Name

	filtered := m.SidebarPanels[:0]
	for _, reg := range m.SidebarPanels {
		if reg.ID != id {
			filtered = append(filtered, reg)
		}
	}
	m.SidebarPanels = filtered

	filteredBottom := m.BottomPanels[:0]
	for _, reg := range m.BottomPanels {
		if reg.ID != id {
			filteredBottom = append(filteredBottom, reg)
		}
	}
	m.BottomPanels = filteredBottom

	if p.SidebarTitle != "" && p.RenderFunc != nil {
		m.SidebarPanels = append(m.SidebarPanels, SidebarRegistration{
			ID:     id,
			Title:  p.SidebarTitle,
			Widget: NewPluginPanelWidget(p, p.RenderFunc, p.EventFunc),
		})
	}
	if p.BottomTitle != "" && p.BottomRenderFunc != nil {
		m.BottomPanels = append(m.BottomPanels, BottomRegistration{
			ID:     id,
			Title:  p.BottomTitle,
			Widget: NewPluginPanelWidget(p, p.BottomRenderFunc, p.BottomEventFunc),
		})
	}
}

func (m *Manager) ApproveAndLoad(p *Plugin) error {
	p.Granted = p.Manifest.Permissions

	m.registry.AddOrUpdate(
		p.Manifest.Name,
		p.Repo,
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

func (m *Manager) RegisterDebugPlugin(p *Plugin) {
	for i, existing := range m.plugins {
		if existing.Name == p.Name {
			existing.Destroy()
			m.plugins[i] = p
			m.collectRegistrations(p)
			return
		}
	}
	m.plugins = append(m.plugins, p)
	m.collectRegistrations(p)
}

func (m *Manager) SetEditorAPI(api EditorAPI) {
	for _, p := range m.plugins {
		p.Editor = api
	}
}

func (m *Manager) SetFilesystemAPI(factory func(pluginDir string) FilesystemAPI) {
	for _, p := range m.plugins {
		p.Filesystem = factory(p.Dir)
	}
}

func (m *Manager) SetSystemAPI(api SystemAPI) {
	for _, p := range m.plugins {
		p.System = api
	}
}

func (m *Manager) SetNetworkAPI(api NetworkAPI) {
	for _, p := range m.plugins {
		p.Network = api
	}
}

func (m *Manager) SetLogFactory(factory func(pluginName string) func(level, message string)) {
	for _, p := range m.plugins {
		p.Host.Log = factory(p.Name)
	}
}

func (m *Manager) DispatchEvent(name string, args ...interface{}) {
	for _, p := range m.plugins {
		if p.State == nil || len(p.EventListeners[name]) == 0 {
			continue
		}
		var largs []lua.LValue
		for _, a := range args {
			switch v := a.(type) {
			case string:
				largs = append(largs, lua.LString(v))
			case int:
				largs = append(largs, lua.LNumber(v))
			case bool:
				largs = append(largs, lua.LBool(v))
			}
		}
		p.DispatchEvent(name, largs...)
	}
}

func (m *Manager) Install(repoURL string) (*Plugin, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return nil, fmt.Errorf("only https:// URLs are allowed for plugin install")
	}
	name := filepath.Base(repoURL)
	name = strings.TrimSuffix(name, ".git")
	if name == "" || name == "." {
		return nil, fmt.Errorf("invalid repository URL")
	}

	targetDir := filepath.Join(m.pluginsDir, name)
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("plugin %q already exists", name)
	}

	if err := m.git.Clone(repoURL, targetDir); err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(targetDir)
	if err != nil {
		os.RemoveAll(targetDir)
		return nil, fmt.Errorf("invalid plugin: %w", err)
	}

	return &Plugin{
		Name:     manifest.Name,
		Dir:      targetDir,
		Repo:     repoURL,
		Manifest: manifest,
	}, nil
}

func (m *Manager) Uninstall(name string) error {
	for i, p := range m.plugins {
		if p.Name == name {
			p.Destroy()
			m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)
			break
		}
	}

	dir := filepath.Join(m.pluginsDir, name)
	if err := os.RemoveAll(dir); err != nil {
		slog.Error("remove plugin directory", "error", err)
	}

	m.registry.Remove(name)
	if err := m.registry.Save(); err != nil {
		slog.Error("save plugin registry", "error", err)
	}

	for i, reg := range m.SidebarPanels {
		if reg.ID == "plugin."+name {
			m.SidebarPanels = append(m.SidebarPanels[:i], m.SidebarPanels[i+1:]...)
			break
		}
	}
	for i, reg := range m.BottomPanels {
		if reg.ID == "plugin."+name {
			m.BottomPanels = append(m.BottomPanels[:i], m.BottomPanels[i+1:]...)
			break
		}
	}

	return nil
}

func (m *Manager) Update(name string) (*Plugin, bool, error) {
	dir := filepath.Join(m.pluginsDir, name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, false, fmt.Errorf("plugin %q not found", name)
	}

	if err := m.git.Pull(dir); err != nil {
		return nil, false, err
	}

	newManifest, err := LoadManifest(dir)
	if err != nil {
		return nil, false, fmt.Errorf("invalid manifest after update: %w", err)
	}

	regEntry := m.registry.Find(name)
	if regEntry == nil {
		p := &Plugin{Name: newManifest.Name, Dir: dir, Manifest: newManifest}
		return p, true, nil
	}

	diff := DiffPermissions(regEntry.Permissions, newManifest.Permissions)
	if !diff.IsEmpty() {
		for i, p := range m.plugins {
			if p.Name == name {
				p.Destroy()
				m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)
				break
			}
		}
		regEntry.Enabled = false
		m.registry.Save()
		p := &Plugin{Name: newManifest.Name, Dir: dir, Manifest: newManifest}
		return p, true, nil
	}

	for _, p := range m.plugins {
		if p.Name == name {
			p.Destroy()
			p.Manifest = newManifest
			p.Granted = regEntry.Permissions
			if err := p.Init(); err != nil {
				return nil, false, err
			}
			m.collectRegistrations(p)
			regEntry.Version = newManifest.Version
			m.registry.Save()
			return p, false, nil
		}
	}

	return nil, false, nil
}

func (m *Manager) SetEnabled(name string, enabled bool) (*Plugin, error) {
	m.registry.SetEnabled(name, enabled)
	if err := m.registry.Save(); err != nil {
		return nil, err
	}

	if !enabled {
		for i, p := range m.plugins {
			if p.Name == name {
				p.Destroy()
				m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)
				break
			}
		}
		return nil, nil
	}

	dir := filepath.Join(m.pluginsDir, name)
	manifest, err := LoadManifest(dir)
	if err != nil {
		return nil, err
	}

	regEntry := m.registry.Find(name)
	if regEntry == nil {
		return nil, fmt.Errorf("plugin %q not in registry", name)
	}

	p := &Plugin{
		Name:     manifest.Name,
		Dir:      dir,
		Manifest: manifest,
		Granted:  regEntry.Permissions,
	}
	if err := p.Init(); err != nil {
		return nil, err
	}

	m.plugins = append(m.plugins, p)
	m.collectRegistrations(p)
	return p, nil
}

func (m *Manager) Reload(name string) (*Plugin, error) {
	var old *Plugin
	var idx int
	for i, p := range m.plugins {
		if p.Name == name {
			old = p
			idx = i
			break
		}
	}
	if old == nil {
		return nil, fmt.Errorf("plugin %q not loaded", name)
	}

	granted := old.Granted
	repo := old.Repo
	dir := old.Dir
	logFn := old.Host.Log

	old.Destroy()

	manifest, err := LoadManifest(dir)
	if err != nil {
		m.plugins = append(m.plugins[:idx], m.plugins[idx+1:]...)
		return nil, fmt.Errorf("reload manifest: %w", err)
	}

	p := &Plugin{
		Name:     manifest.Name,
		Dir:      dir,
		Repo:     repo,
		Manifest: manifest,
		Granted:  granted,
		Host:     HostCallbacks{Log: logFn},
	}

	if err := p.Init(); err != nil {
		m.plugins = append(m.plugins[:idx], m.plugins[idx+1:]...)
		return nil, fmt.Errorf("reload init: %w", err)
	}

	m.plugins[idx] = p

	for i := len(m.SidebarPanels) - 1; i >= 0; i-- {
		if m.SidebarPanels[i].ID == "plugin."+name {
			m.SidebarPanels = append(m.SidebarPanels[:i], m.SidebarPanels[i+1:]...)
		}
	}
	for i := len(m.BottomPanels) - 1; i >= 0; i-- {
		if m.BottomPanels[i].ID == "plugin."+name {
			m.BottomPanels = append(m.BottomPanels[:i], m.BottomPanels[i+1:]...)
		}
	}
	m.collectRegistrations(p)

	return p, nil
}

func (m *Manager) PluginsDir() string {
	return m.pluginsDir
}

func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) InstalledPluginNames() []string {
	if m.registry == nil {
		return nil
	}
	var names []string
	for _, e := range m.registry.Entries {
		names = append(names, e.Name)
	}
	return names
}

func (m *Manager) FindPlugin(name string) *Plugin {
	for _, p := range m.plugins {
		if p.Name == name {
			return p
		}
	}
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
