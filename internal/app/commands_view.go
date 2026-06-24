package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) ToggleTerminal() {
	if !a.ContentSplit.ShowBottom {
		r := a.ContentSplit.GetRect()
		maxH := r.H - 4
		if a.ContentSplit.BottomH <= 1 || a.ContentSplit.BottomH > maxH {
			a.ContentSplit.BottomH = min(r.H/2, maxH)
		}
		a.showTerminalPanel()
	} else {
		a.HideBottomPanel()
	}
}

func (a *App) ToggleTerminalFullscreen() {
	r := a.ContentSplit.GetRect()
	fullH := r.H - 1
	if a.ContentSplit.ShowBottom && a.ContentSplit.BottomH >= fullH {
		a.HideBottomPanel()
	} else {
		a.ContentSplit.BottomH = fullH
		a.showTerminalPanel()
	}
}

func (a *App) FocusPanel() {
	if !a.ContentSplit.ShowBottom {
		a.ContentSplit.ShowBottom = true
	}
	if w := a.BottomPanel.ActiveWidget(); w != nil {
		a.Root.SetFocus(w)
	}
}

func (a *App) ShowKeybindings() {
	defaults := make(map[string]string)
	for _, kb := range config.DefaultKeybindings() {
		defaults[kb.Command] = FormatKeyBinding(kb.Key)
	}

	w := ui.NewKeybindingsWidget(a.Reg.List())
	w.Borders = a.Borders
	w.GetShortcut = func(cmdID string) string {
		for _, kb := range a.Keybindings {
			if kb.Command == cmdID {
				return FormatKeyBinding(kb.Key)
			}
		}
		return ""
	}
	w.GetDefault = func(cmdID string) string {
		return defaults[cmdID]
	}
	w.OnEdit = func(cmdID string, newKey string) {
		filtered := make([]config.KeyBinding, 0, len(a.Keybindings))
		for _, kb := range a.Keybindings {
			if kb.Key != newKey && kb.Command != cmdID {
				filtered = append(filtered, kb)
			}
		}
		filtered = append(filtered, config.KeyBinding{Key: newKey, Command: cmdID})
		a.Keybindings = filtered
		config.SaveKeybindings(a.Keybindings)
		a.RebindKeys()
	}
	w.OnReset = func(cmdID string) {
		var defaultKey string
		for _, kb := range config.DefaultKeybindings() {
			if kb.Command == cmdID {
				defaultKey = kb.Key
				break
			}
		}
		filtered := make([]config.KeyBinding, 0, len(a.Keybindings))
		for _, kb := range a.Keybindings {
			if kb.Command == cmdID {
				continue
			}
			if defaultKey != "" && kb.Key == defaultKey {
				continue
			}
			filtered = append(filtered, kb)
		}
		if defaultKey != "" {
			filtered = append(filtered, config.KeyBinding{Key: defaultKey, Command: cmdID})
		}
		a.Keybindings = filtered
		config.SaveKeybindings(a.Keybindings)
		a.RebindKeys()
	}
	w.OnClear = func(cmdID string) {
		filtered := make([]config.KeyBinding, 0, len(a.Keybindings))
		for _, kb := range a.Keybindings {
			if kb.Command != cmdID {
				filtered = append(filtered, kb)
			}
		}
		a.Keybindings = filtered
		config.SaveKeybindings(a.Keybindings)
		a.RebindKeys()
	}
	w.OnHelp = func() {
		help := ui.NewInfoDialogWidget("Keyboard Shortcuts Help", []ui.InfoEntry{
			{Key: "Enter", Desc: "Edit selected shortcut"},
			{Key: "Backspace", Desc: "Reset to default"},
			{Key: "Delete", Desc: "Clear shortcut"},
			{Key: "Up/Down", Desc: "Navigate list"},
			{Key: "Esc", Desc: "Close"},
		})
		help.Borders = a.Borders
		help.OnDismiss = func() {
			a.Root.PopOverlay()
			a.Root.SetFocus(w)
		}
		a.Root.PushOverlay(ui.Overlay{Widget: help, Modal: true})
		a.Root.SetFocus(help)
	}
	w.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(w)
}

func (a *App) ToggleSearchReplace() {
	if a.Sidebar.Visible && a.Sidebar.ActivePanel == "search" {
		a.Search.ToggleReplaceMode()
	} else {
		a.Search.SetReplaceMode(true)
		a.ShowPanel("search", a.Search)
	}
	a.Root.SetFocus(a.Search)
}

func registerViewCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "sidebar.toggle", Title: "Toggle Sidebar",
		Keywords: []string{"view", "panel", "show", "hide"},
		Handler:  app.ToggleSidebar,
	})

	reg.Register(command.Command{
		ID: "sidebar.explorer", Title: "Show Explorer",
		Keywords: []string{"view", "file", "tree", "browser"},
		Handler: func() {
			app.Explorer.Reload()
			app.ShowPanel("explorer", app.Explorer)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.search", Title: "Show Search",
		Keywords: []string{"view", "search", "find", "grep"},
		Handler: func() {
			app.ShowPanel("search", app.Search)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.searchReplace", Title: "Search and Replace in Files",
		Keywords: []string{"view", "search", "find", "replace", "substitute"},
		Handler:  app.ToggleSearchReplace,
	})

	reg.Register(command.Command{
		ID: "sidebar.changes", Title: "Show Changes",
		Keywords: []string{"view", "git", "diff", "source control"},
		Handler: func() {
			app.Changes.Refresh()
			app.ShowPanel("changes", app.Changes)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.widgets", Title: "Show Widgets",
		Keywords: []string{"view", "widget", "plugin", "tree"},
		Handler: func() {
			app.ShowPanel("widgets", app.WidgetPanel)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.reloadWidgets", Title: "Reload Widgets",
		Keywords: []string{"widget", "reload", "refresh", "json"},
		Handler: func() {
			app.ReloadWidgetPanel()
			app.ShowPanel("widgets", app.WidgetPanel)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.wider", Title: "Increase Sidebar Width",
		Keywords: []string{"view", "resize"},
		Handler: func() {
			app.SetSidebarWidth(app.SplitPanel.DividerPos + 1)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.narrower", Title: "Decrease Sidebar Width",
		Keywords: []string{"view", "resize"},
		Handler: func() {
			if app.Sidebar.Visible {
				app.SetSidebarWidth(app.SplitPanel.DividerPos - 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.focus", Title: "Focus Sidebar",
		Keywords: []string{"view"},
		Handler:  app.FocusSidebar,
	})

	reg.Register(command.Command{
		ID: "panel.toggle", Title: "Toggle Panel",
		Keywords: []string{"view", "bottom", "show", "hide"},
		Handler:  app.ToggleBottomPanel,
	})

	reg.Register(command.Command{
		ID: "panel.focus", Title: "Focus Panel",
		Keywords: []string{"view", "bottom"},
		Handler:  app.FocusPanel,
	})

	reg.Register(command.Command{
		ID: "panel.taller", Title: "Increase Panel Height",
		Keywords: []string{"view", "resize", "bottom"},
		Handler: func() {
			if !app.ContentSplit.ShowBottom {
				app.ShowBottomPanel()
			}
			app.ContentSplit.BottomH++
		},
	})

	reg.Register(command.Command{
		ID: "panel.shorter", Title: "Decrease Panel Height",
		Keywords: []string{"view", "resize", "bottom"},
		Handler: func() {
			if app.ContentSplit.ShowBottom && app.ContentSplit.BottomH > 1 {
				app.ContentSplit.BottomH--
			}
		},
	})

	reg.Register(command.Command{
		ID: "terminal.new", Title: "New Terminal",
		Keywords: []string{"terminal", "shell", "console", "bash"},
		Handler:  app.SpawnTerminal,
	})

	reg.Register(command.Command{
		ID: "terminal.toggle", Title: "Toggle Terminal",
		Keywords: []string{"terminal", "shell", "console", "bash"},
		Handler:  app.ToggleTerminal,
	})

	reg.Register(command.Command{
		ID: "terminal.fullscreen", Title: "Toggle Terminal Fullscreen",
		Keywords: []string{"terminal", "shell", "maximize"},
		Handler:  app.ToggleTerminalFullscreen,
	})

	reg.Register(command.Command{
		ID: "terminal.closeAll", Title: "Close All Terminals",
		Keywords: []string{"terminal", "shell"},
		Handler:  app.CloseAllTerminals,
	})

	reg.Register(command.Command{
		ID: "view.keybindings", Title: "Keyboard Shortcuts",
		Keywords: []string{"view", "keybindings", "shortcuts", "remap", "rebind", "hotkeys"},
		Handler:  app.ShowKeybindings,
	})

	reg.Register(command.Command{
		ID: "about", Title: "About TTT Editor",
		Keywords: []string{"help", "version", "info"},
		Handler: func() {
			dialog := ui.NewInfoDialogWidget("About TTT Editor", []ui.InfoEntry{
				{Key: "Version", Desc: app.Version},
				{Key: "Website", Desc: "https://tttedit.dev"},
				{Key: "GitHub", Desc: "https://github.com/eugenioenko/ttt"},
			})
			dialog.Borders = app.Borders
			dialog.InvertStyles = true
			dialog.OnDismiss = func() {
				app.DismissDialog()
			}
			app.ShowDialog(dialog)
		},
	})
}
