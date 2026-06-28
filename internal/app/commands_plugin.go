package app

import (
	"log/slog"
	"time"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/markdown"
	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

func registerPluginCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID:       "plugin.list",
		Title:    "Plugins: List Installed",
		Keywords: []string{"plugin", "extension"},
		Handler:  func() { app.showPluginList() },
	})

	reg.Register(command.Command{
		ID:       "plugin.install",
		Title:    "Plugins: Install from URL",
		Keywords: []string{"plugin", "extension", "install"},
		Handler:  func() { app.pluginInstall() },
	})

	reg.Register(command.Command{
		ID:       "plugin.uninstall",
		Title:    "Plugins: Uninstall",
		Keywords: []string{"plugin", "extension", "remove"},
		Handler:  func() { app.pluginUninstall() },
	})

	reg.Register(command.Command{
		ID:       "plugin.update",
		Title:    "Plugins: Update",
		Keywords: []string{"plugin", "extension", "upgrade"},
		Handler:  func() { app.pluginUpdate() },
	})

	reg.Register(command.Command{
		ID:       "plugin.showPanel",
		Title:    "Plugins: Show Panel",
		Keywords: []string{"plugin", "extension", "panel"},
		Handler: func() {
			app.Sidebar.SetActivePanel("plugins")
			app.SplitPanel.ShowLeft = true
		},
	})

	reg.Register(command.Command{
		ID:       "plugin.reload",
		Title:    "Plugins: Reload",
		Keywords: []string{"plugin", "reload", "refresh"},
		Handler:  func() { app.pluginReload() },
	})

	reg.Register(command.Command{
		ID:       "plugin.reloadAll",
		Title:    "Plugins: Reload All",
		Keywords: []string{"plugin", "reload", "refresh", "all"},
		Handler:  func() { app.pluginReloadAll() },
	})

	reg.Register(command.Command{
		ID:       "plugin.clearOutput",
		Title:    "Plugins: Clear Output",
		Keywords: []string{"plugin", "output", "clear", "log"},
		Handler:  func() { app.Output.Clear() },
	})
}

func (a *App) showPluginList() {
	plugins := a.PluginManager.Plugins()
	if len(plugins) == 0 {
		a.ShowConfirmDialogEx("Plugins", "No plugins installed.", []string{"Close"}, []func(){
			func() { a.DismissDialog() },
		})
		return
	}

	var entries []widgets.KeyValueEntry
	for _, p := range plugins {
		status := "disabled"
		if p.Enabled {
			status = "enabled"
		}
		if p.LastError != nil {
			status = "error"
		}
		entries = append(entries, widgets.KeyValueEntry{
			Key:   p.Name,
			Value: status + " v" + p.Manifest.Version,
		})
	}

	a.ShowInfoDialog("Installed Plugins", entries)
}

func (a *App) pluginInstall() {
	a.ShowInputDialogEx("Install Plugin", "Repository URL", "", "Install", func(repoURL string) {
		go func() {
			p, err := a.PluginManager.Install(repoURL)
			a.Screen.PostEvent(tcell.NewEventInterrupt(&pluginInstallResult{
				plugin: p,
				err:    err,
			}))
		}()
	})
}

type pluginInstallResult struct {
	plugin *plugin.Plugin
	err    error
}

func (a *App) handlePluginInstallResult(result *pluginInstallResult) {
	if result.err != nil {
		a.ShowConfirmDialogEx("Install Failed", result.err.Error(), []string{"Close"}, []func(){
			func() { a.DismissDialog() },
		})
		return
	}
	a.PendingPluginApprovals = append(a.PendingPluginApprovals, result.plugin)
	if len(a.PendingPluginApprovals) == 1 {
		a.ShowPluginApprovalDialog(result.plugin)
	}
}

func (a *App) pluginUninstall() {
	names := a.PluginManager.InstalledPluginNames()
	if len(names) == 0 {
		a.ShowConfirmDialogEx("Uninstall", "No plugins installed.", []string{"Close"}, []func(){
			func() { a.DismissDialog() },
		})
		return
	}

	var items []widgets.SelectItem
	for _, name := range names {
		items = append(items, widgets.SelectItem{ID: name, Label: name})
	}

	a.ShowSelectDialog("Uninstall Plugin", items, func(name string) {
		a.ShowConfirmDialogEx("Confirm Uninstall", "Remove plugin \""+name+"\"?", []string{"Cancel", "Uninstall"}, []func(){
			func() { a.DismissDialog() },
			func() {
				a.DismissDialog()
				a.doPluginUninstall(name)
			},
		})
	}, nil)
}

func (a *App) doPluginUninstall(name string) {
	a.Sidebar.RemovePanel("plugin." + name)
	a.BottomPanel.RemovePanel("plugin." + name)

	if err := a.PluginManager.Uninstall(name); err != nil {
		slog.Error("plugin uninstall", "error", err)
	}

	if a.PluginsPanel != nil {
		a.PluginsPanel.Refresh()
	}
}

func (a *App) pluginUpdate() {
	names := a.PluginManager.InstalledPluginNames()
	if len(names) == 0 {
		a.ShowConfirmDialogEx("Update", "No plugins installed.", []string{"Close"}, []func(){
			func() { a.DismissDialog() },
		})
		return
	}

	var items []widgets.SelectItem
	for _, name := range names {
		items = append(items, widgets.SelectItem{ID: name, Label: name})
	}

	a.ShowSelectDialog("Update Plugin", items, func(name string) {
		go func() {
			p, needsApproval, err := a.PluginManager.Update(name)
			a.Screen.PostEvent(tcell.NewEventInterrupt(&pluginUpdateResult{
				plugin:        p,
				needsApproval: needsApproval,
				err:           err,
				name:          name,
			}))
		}()
	}, nil)
}

type pluginUpdateResult struct {
	plugin        *plugin.Plugin
	needsApproval bool
	err           error
	name          string
}

func (a *App) handlePluginUpdateResult(result *pluginUpdateResult) {
	if result.err != nil {
		a.ShowConfirmDialogEx("Update Failed", result.err.Error(), []string{"Close"}, []func(){
			func() { a.DismissDialog() },
		})
		return
	}
	if result.needsApproval && result.plugin != nil {
		a.PendingPluginApprovals = append(a.PendingPluginApprovals, result.plugin)
		if len(a.PendingPluginApprovals) == 1 {
			a.ShowPluginApprovalDialog(result.plugin)
		}
		return
	}
	if !result.needsApproval && result.plugin != nil {
		a.WirePlugin(result.plugin)
	}
	if a.PluginsPanel != nil {
		a.PluginsPanel.Refresh()
	}
}

func (a *App) ShowPluginApprovalDialog(p *plugin.Plugin) {
	permEntries := p.Manifest.Permissions.DisplayEntries()

	var kvEntries []widgets.KeyValueEntry
	for _, e := range permEntries {
		kvEntries = append(kvEntries, widgets.KeyValueEntry{
			Key:   e.Name,
			Value: e.Value,
		})
	}

	content := widgets.NewKeyValueListWidget(kvEntries)
	content.InvertStyles = true

	dialog := widgets.NewDialogWidget(50)
	dialog.Title = p.Manifest.Name + " requests"
	dialog.Borders = *a.Borders
	dialog.SetContent(content)
	dialog.Buttons = []widgets.DialogButton{
		{Label: "&Cancel", Handler: func() {
			a.DismissDialog()
			a.showNextPluginApproval()
		}},
		{Label: "&Allow", Handler: func() {
			a.DismissDialog()
			if err := a.PluginManager.ApproveAndLoad(p); err == nil {
				a.WirePlugin(p)
			}
			if a.PluginsPanel != nil {
				a.PluginsPanel.Refresh()
			}
			a.showNextPluginApproval()
		}},
	}
	dialog.OnDismiss = func() {
		a.DismissDialog()
		a.showNextPluginApproval()
	}
	dialog.Build()

	adapter := ui.NewWidgetAdapter(dialog)
	a.ShowDialog(adapter)
}

func (a *App) WirePlugin(p *plugin.Plugin) {
	for _, reg := range a.PluginManager.SidebarPanels {
		if reg.ID == "plugin."+p.Name {
			if !a.Sidebar.HasPanel(reg.ID) {
				a.Sidebar.AddPanel(reg.ID, reg.Title, ui.NewWidgetAdapter(reg.Widget))
			}
			break
		}
	}
	for _, reg := range a.PluginManager.BottomPanels {
		if reg.ID == "plugin."+p.Name {
			if !a.BottomPanel.HasPanel(reg.ID) {
				a.BottomPanel.AddPanel(reg.ID, reg.Title, ui.NewWidgetAdapter(reg.Widget))
			}
			break
		}
	}
	p.RequestRedraw = func() {
		a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
	p.SimulateClick = func(x, y int) {
		go func() {
			a.Screen.PostEvent(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone))
			time.Sleep(50 * time.Millisecond)
			a.Screen.PostEvent(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone))
		}()
	}
	p.SimulateDrag = func(x1, y1, x2, y2 int) {
		go func() {
			a.Screen.PostEvent(tcell.NewEventMouse(x1, y1, tcell.Button1, tcell.ModNone))
			time.Sleep(30 * time.Millisecond)
			steps := 10
			for i := 1; i <= steps; i++ {
				mx := x1 + (x2-x1)*i/steps
				my := y1 + (y2-y1)*i/steps
				a.Screen.PostEvent(tcell.NewEventMouse(mx, my, tcell.Button1, tcell.ModNone))
				time.Sleep(15 * time.Millisecond)
			}
			a.Screen.PostEvent(tcell.NewEventMouse(x2, y2, tcell.ButtonNone, tcell.ModNone))
		}()
	}
	p.ScreenshotToFile = func(path string) error {
		return a.DumpScreenshot(path)
	}
	p.DebugDumpToFile = func(path string) error {
		return a.DumpDebugState(path)
	}
	p.QuitApp = func() {
		if a.Running != nil {
			*a.Running = false
		}
		a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
	p.PostAsync = func(result *plugin.PluginAsyncResult) {
		a.Screen.PostEvent(tcell.NewEventInterrupt(result))
	}
	p.RenderMarkdown = func(text string) []plugin.MarkdownLine {
		rendered := markdown.Render(text)
		lines := make([]plugin.MarkdownLine, len(rendered))
		for i, line := range rendered {
			spans := make([]plugin.MarkdownSpan, len(line.Spans))
			for j, span := range line.Spans {
				spans[j] = plugin.MarkdownSpan{Text: span.Text, Style: span.Style}
			}
			lines[i] = plugin.MarkdownLine{Spans: spans}
		}
		return lines
	}
	p.Editor = NewPluginEditorAPI(a)
	fsRoots := a.Workspace.Paths()
	if p.Dir != "" {
		fsRoots = append(fsRoots, p.Dir)
	}
	p.Filesystem = NewPluginFilesystemAPI(fsRoots...)
	p.System = NewPluginSystemAPI()
	p.Network = NewPluginNetworkAPI()
	a.wirePluginLog(p)
	p.Borders = a.Borders
	p.ShowInfoDialog = func(title string, entries []widgets.KeyValueEntry) {
		a.ShowInfoDialog(title, entries)
	}
	p.ShowConfirmDialog = func(message string, onConfirm func()) {
		a.ShowConfirmDialogEx("Confirm", message, []string{"Cancel", "OK"}, []func(){
			func() { a.DismissDialog() },
			func() {
				a.DismissDialog()
				onConfirm()
			},
		})
	}
	p.ShowContextMenu = func(entries []widgets.MenuEntry, x, y int, onCommand func(string)) {
		items := make([]ui.ContextMenuItem, len(entries))
		for i, e := range entries {
			items[i] = ui.ContextMenuItem{
				Label:   e.Label,
				Command: e.Command,
				IsSep:   e.Separator,
			}
		}
		menu := ui.NewContextMenuWidget(items, x, y)
		menu.Borders = a.Borders
		menu.OnExec = func(cmd string) {
			a.Root.PopOverlay()
			onCommand(cmd)
		}
		menu.OnDismiss = func() {
			a.Root.PopOverlay()
		}
		a.Root.PushOverlay(ui.Overlay{Widget: menu, Modal: true})
		a.Root.SetFocus(menu)
	}

	p.OpenDrawer = func(panel *plugin.PluginPanelWidget, width, minWidth int) {
		drawer := widgets.NewDrawerWidget(widgets.DrawerConfig{
			Width:    width,
			MinWidth: minWidth,
			Borders:  *a.Borders,
			OnDismiss: func() {
				a.DismissDialog()
			},
		})
		drawer.SetContent(panel)
		a.ShowDrawer(drawer)
	}
	p.CloseDrawer = func() {
		a.DismissDialog()
	}
	p.OpenTab = func(id string, panel *plugin.PluginPanelWidget) {
		a.EditorGroup.OpenPluginTab(id, panel)
	}
	p.CloseTab = func(id string) {
		a.EditorGroup.ClosePluginTab(id)
	}

	if p.HasSidebarMenu() {
		for _, entry := range p.SidebarMenuEntries {
			if entry.Separator || entry.Command == "" {
				continue
			}
			cmd := entry.Command
			a.Reg.Register(command.Command{
				ID:    cmd,
				Title: entry.Label,
				Handler: func() {
					p.CallSidebarAction(cmd)
				},
			})
		}
	}

	a.registerPluginCommandsAndKeys(p)
}

func (a *App) wirePluginLog(p *plugin.Plugin) {
	p.Log = func(level, message string) {
		a.Output.AddLine(ui.OutputLine{
			Time:       time.Now().Format("15:04:05"),
			PluginName: p.Name,
			Level:      level,
			Message:    message,
		})
		a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
}

func (a *App) registerPluginCommandsAndKeys(p *plugin.Plugin) {
	for _, cmd := range p.Commands {
		handler := cmd.Handler
		cmdID := cmd.ID
		plugName := p.Name
		a.Reg.Register(command.Command{
			ID:    cmd.ID,
			Title: cmd.Title,
			Handler: func() {
				if err := handler(); err != nil {
					slog.Error("plugin command error", "plugin", plugName, "command", cmdID, "error", err)
				}
			},
		})
	}

	for _, kb := range p.PluginKeybindings {
		steps, err := config.ParseKeyString(kb.Key)
		if err != nil {
			slog.Warn("plugin keybinding parse error", "plugin", p.Name, "key", kb.Key, "error", err)
			continue
		}
		cmdID := kb.Command
		if len(steps) > 1 {
			tcellSteps := make([]ui.GlobalKeyBinding, len(steps))
			for i, step := range steps {
				key, mod, rn := comboToTcell(step)
				tcellSteps[i] = ui.GlobalKeyBinding{Key: key, Mod: mod, Rune: rn}
			}
			a.Root.AddChordKey(tcellSteps, func() {
				a.Reg.Execute(cmdID)
			})
		} else {
			key, mod, rn := comboToTcell(steps[0])
			a.Root.AddGlobalKey(key, mod, rn, func() {
				a.Reg.Execute(cmdID)
			})
		}
		a.Reg.SetShortcut(cmdID, FormatKeyBinding(kb.Key))
	}
}

func (a *App) RegisterStartupPluginCommands() {
	for _, p := range a.PluginManager.Plugins() {
		a.WirePlugin(p)
	}
}

func (a *App) pluginReload() {
	plugins := a.PluginManager.Plugins()
	if len(plugins) == 0 {
		a.ShowConfirmDialogEx("Reload", "No plugins loaded.", []string{"Close"}, []func(){
			func() { a.DismissDialog() },
		})
		return
	}

	if len(plugins) == 1 {
		a.doPluginReload(plugins[0].Name)
		return
	}

	var items []widgets.SelectItem
	for _, p := range plugins {
		items = append(items, widgets.SelectItem{ID: p.Name, Label: p.Name})
	}
	a.ShowSelectDialog("Reload Plugin", items, func(name string) {
		a.doPluginReload(name)
	}, nil)
}

func (a *App) doPluginReload(name string) {
	a.Sidebar.RemovePanel("plugin." + name)
	a.BottomPanel.RemovePanel("plugin." + name)

	p, err := a.PluginManager.Reload(name)
	if err != nil {
		a.Output.AddLine(ui.OutputLine{
			Time:       time.Now().Format("15:04:05"),
			PluginName: name,
			Level:      "error",
			Message:    "reload failed: " + err.Error(),
		})
		if a.PluginsPanel != nil {
			a.PluginsPanel.Refresh()
		}
		return
	}

	a.WirePlugin(p)

	a.Output.AddLine(ui.OutputLine{
		Time:       time.Now().Format("15:04:05"),
		PluginName: name,
		Level:      "info",
		Message:    "reloaded successfully",
	})
	if a.PluginsPanel != nil {
		a.PluginsPanel.Refresh()
	}
}

func (a *App) pluginReloadAll() {
	plugins := a.PluginManager.Plugins()
	if len(plugins) == 0 {
		return
	}

	var names []string
	for _, p := range plugins {
		names = append(names, p.Name)
	}
	for _, name := range names {
		a.doPluginReload(name)
	}
}

type RemoteRegistryResult struct {
	Entries []plugin.RemoteRegistryEntry
}

func (a *App) PluginInstallFromURL(repoURL string) {
	go func() {
		p, err := a.PluginManager.Install(repoURL)
		a.Screen.PostEvent(tcell.NewEventInterrupt(&pluginInstallResult{
			plugin: p,
			err:    err,
		}))
	}()
}

func (a *App) PluginUninstallByName(name string) {
	a.ShowConfirmDialogEx("Confirm Uninstall", "Remove plugin \""+name+"\"?", []string{"Cancel", "Uninstall"}, []func(){
		func() { a.DismissDialog() },
		func() {
			a.DismissDialog()
			a.doPluginUninstall(name)
		},
	})
}

func (a *App) PluginUpdateByName(name string) {
	go func() {
		p, needsApproval, err := a.PluginManager.Update(name)
		a.Screen.PostEvent(tcell.NewEventInterrupt(&pluginUpdateResult{
			plugin:        p,
			needsApproval: needsApproval,
			err:           err,
			name:          name,
		}))
	}()
}

func (a *App) handleRemoteRegistryResult(result *RemoteRegistryResult) {
	if a.PluginsPanel != nil {
		a.PluginsPanel.SetAvailable(result.Entries)
	}
}

func (a *App) showNextPluginApproval() {
	if len(a.PendingPluginApprovals) > 0 {
		a.PendingPluginApprovals = a.PendingPluginApprovals[1:]
	}
	if len(a.PendingPluginApprovals) > 0 {
		a.ShowPluginApprovalDialog(a.PendingPluginApprovals[0])
	}
}

func (a *App) ShowPendingPluginApprovals() {
	if len(a.PendingPluginApprovals) > 0 {
		a.ShowPluginApprovalDialog(a.PendingPluginApprovals[0])
	}
}
