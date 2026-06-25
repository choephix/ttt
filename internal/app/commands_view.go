package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
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
		content := widgets.NewKeyValueListWidget([]widgets.KeyValueEntry{
			{Key: "Enter", Value: "Edit selected shortcut"},
			{Key: "Backspace", Value: "Reset to default"},
			{Key: "Delete", Value: "Clear shortcut"},
			{Key: "Up/Down", Value: "Navigate list"},
			{Key: "Esc", Value: "Close"},
		})
		dialog := widgets.NewDialogWidget(50)
		dialog.Title = "Keyboard Shortcuts Help"
		dialog.Borders = *a.Borders
		dialog.SetContent(content)
		dialog.Buttons = []widgets.DialogButton{
			{Label: "&Close", Handler: func() {
				a.Root.PopOverlay()
				a.Root.SetFocus(w)
			}},
		}
		dialog.OnDismiss = func() {
			a.Root.PopOverlay()
			a.Root.SetFocus(w)
		}
		dialog.Build()
		adapter := ui.NewWidgetAdapter(dialog)
		a.Root.PushOverlay(ui.Overlay{Widget: adapter, Modal: true})
		a.Root.SetFocus(adapter)
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
		ID: "sidebar.navigation", Title: "Show Navigation",
		Keywords: []string{"view", "file", "tree", "browse", "navigate"},
		Handler: func() {
			app.Navigation.Reload()
			app.ShowPanel("navigation", app.Navigation.Adapter)
		},
	})

	reg.Register(command.Command{
		ID: "navigate.open", Title: "Navigate: Open File",
		Handler: func() {
			if node := app.NavigationContextNode; node != nil {
				app.EditorGroup.OpenFile(node.ID)
				app.FocusEditorIfEnabled()
			}
		},
	})

	reg.Register(command.Command{
		ID: "navigate.refresh", Title: "Navigate: Refresh",
		Handler: func() {
			app.Navigation.Reload()
		},
	})

	reg.Register(command.Command{
		ID: "navigate.newFile", Title: "Navigate: New File",
		Handler: func() { app.NavigateNewFile() },
	})
	reg.Register(command.Command{
		ID: "navigate.newFolder", Title: "Navigate: New Folder",
		Handler: func() { app.NavigateNewFolder() },
	})
	reg.Register(command.Command{
		ID: "navigate.rename", Title: "Navigate: Rename",
		Handler: func() { app.NavigateRename() },
	})
	reg.Register(command.Command{
		ID: "navigate.delete", Title: "Navigate: Delete",
		Handler: func() { app.NavigateDelete() },
	})
	reg.Register(command.Command{
		ID: "navigate.copyAbsolutePath", Title: "Navigate: Copy Absolute Path",
		Handler: func() { app.NavigateCopyAbsolutePath() },
	})
	reg.Register(command.Command{
		ID: "navigate.copyRelativePath", Title: "Navigate: Copy Relative Path",
		Handler: func() { app.NavigateCopyRelativePath() },
	})
	reg.Register(command.Command{
		ID: "navigate.removeRoot", Title: "Navigate: Remove from Workspace",
		Handler: func() { app.NavigateRemoveRoot() },
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
			app.ShowInfoDialogEx("About TTT Editor", []widgets.KeyValueEntry{
				{Key: "Version", Value: app.Version},
				{Key: "Website", Value: "https://tttedit.dev"},
				{Key: "GitHub", Value: "https://github.com/eugenioenko/ttt"},
			}, true)
		},
	})

	reg.Register(command.Command{
		ID: "drawer.open", Title: "Open Drawer",
		Keywords: []string{"view", "panel", "drawer", "right"},
		Handler: func() {
			if app.Root.HasOverlay() {
				return
			}
			drawer := widgets.NewDrawerWidget(widgets.DrawerConfig{
				Width:   40,
				Borders: *app.Borders,
				OnDismiss: func() {
					app.DismissDialog()
				},
			})

			title := widgets.NewTitleWidget(widgets.TitleConfig{Title: "Widget Demo"})
			title.Box.PaddingLeft = 1
			title.Box.PaddingRight = 1

			nameInput := widgets.NewInputWidget(widgets.InputConfig{
				Placeholder: "Enter your name",
				Bordered:    true,
			})
			nameInput.Box.PaddingLeft = 1
			nameInput.Box.PaddingRight = 1

			emailInput := widgets.NewInputWidget(widgets.InputConfig{
				Placeholder: "Enter your email",
				Bordered:    true,
			})
			emailInput.Box.PaddingLeft = 1
			emailInput.Box.PaddingRight = 1

			darkMode := widgets.NewCheckboxWidget(widgets.CheckboxConfig{
				Label:   "Dark Mode",
				Checked: true,
			})
			darkMode.Box.PaddingLeft = 1

			notifications := widgets.NewCheckboxWidget(widgets.CheckboxConfig{
				Label: "Enable Notifications",
			})
			notifications.Box.PaddingLeft = 1

			sel := widgets.NewSelectWidget(widgets.SelectConfig{
				Items: []widgets.SelectItem{
					{ID: "small", Label: "Small"},
					{ID: "medium", Label: "Medium"},
					{ID: "large", Label: "Large"},
				},
				Collapsible: true,
			})
			sel.Box.PaddingLeft = 1
			sel.Box.PaddingRight = 1

			saveBtn := widgets.NewButtonWidget(widgets.ButtonConfig{
				Label: "Save",
				OnClick: func() {
					app.DismissDialog()
				},
			})

			cancelBtn := widgets.NewButtonWidget(widgets.ButtonConfig{
				Label: "Cancel",
				OnClick: func() {
					app.DismissDialog()
				},
			})

			footer := widgets.NewHStackWidget(saveBtn, cancelBtn)
			footer.Align = "right"
			footer.Gap = 1
			footer.Box.PaddingRight = 1

			sizeLabel := widgets.NewLabelWidget(widgets.LabelConfig{Text: "Size:"})
			sizeLabel.Box.PaddingLeft = 1

			content := widgets.NewVStackWidget(
				title,
				widgets.NewDividerWidget(widgets.DividerConfig{}),
				nameInput,
				emailInput,
				widgets.NewDividerWidget(widgets.DividerConfig{}),
				darkMode,
				notifications,
				widgets.NewDividerWidget(widgets.DividerConfig{}),
				sizeLabel,
				sel,
				widgets.NewDividerWidget(widgets.DividerConfig{}),
				footer,
			)

			drawer.SetContent(content)
			app.ShowDrawer(drawer)
		},
	})
}
