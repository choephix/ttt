package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) ToggleTerminal() {
	if !a.ContentSplit.ShowBottom {
		r := a.ContentSplit.GetRect()
		half := r.H / 2
		if half > r.H-4 {
			half = r.H - 4
		}
		a.ContentSplit.BottomH = half
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

func (a *App) ShowKeyboardTester() {
	kt := ui.NewKeyTesterWidget()
	kt.Borders = a.Borders
	kt.LookupBinding = func(combo string) string {
		for _, kb := range a.Keybindings {
			if kb.Key == combo {
				return kb.Command
			}
		}
		return ""
	}
	kt.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(kt)
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
		ID: "view.keyboardTester", Title: "Keyboard Tester",
		Keywords: []string{"view", "debug", "input"},
		Handler:  app.ShowKeyboardTester,
	})

	reg.Register(command.Command{
		ID: "about", Title: "About ttt",
		Keywords: []string{"help", "version", "info"},
		Handler: func() {
			OpenURL("https://github.com/eugenioenko/ttt")
		},
	})
}
