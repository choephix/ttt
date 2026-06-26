package app

import (
	"github.com/eugenioenko/ttt/internal/command"
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
				for _, reg := range a.PluginManager.SidebarPanels {
					if reg.ID == "plugin."+p.Name {
						a.Sidebar.AddPanel(reg.ID, reg.Title, ui.NewWidgetAdapter(reg.Widget))
						break
					}
				}
				for _, reg := range a.PluginManager.BottomPanels {
					if reg.ID == "plugin."+p.Name {
						a.BottomPanel.AddPanel(reg.ID, reg.Title, ui.NewWidgetAdapter(reg.Widget))
						break
					}
				}
				p.RequestRedraw = func() {
					a.Screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
				p.PostAsync = func(result *plugin.PluginAsyncResult) {
					a.Screen.PostEvent(tcell.NewEventInterrupt(result))
				}
				p.Editor = NewPluginEditorAPI(a)
				p.Filesystem = NewPluginFilesystemAPI()
				p.System = NewPluginSystemAPI()
				p.Network = NewPluginNetworkAPI()
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
