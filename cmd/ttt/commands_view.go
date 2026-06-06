package main

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) ToggleTerminal() {
	if !a.contentSplit.ShowBottom {
		r := a.contentSplit.GetRect()
		half := r.H / 2
		if half > r.H-4 {
			half = r.H - 4
		}
		a.contentSplit.BottomH = half
		a.showTerminalPanel()
	} else {
		a.HideBottomPanel()
	}
}

func (a *App) ToggleTerminalFullscreen() {
	r := a.contentSplit.GetRect()
	fullH := r.H - 1
	if a.contentSplit.ShowBottom && a.contentSplit.BottomH >= fullH {
		a.HideBottomPanel()
	} else {
		a.contentSplit.BottomH = fullH
		a.showTerminalPanel()
	}
}

func (a *App) FocusPanel() {
	if !a.contentSplit.ShowBottom {
		a.contentSplit.ShowBottom = true
	}
	if w := a.bottomPanel.ActiveWidget(); w != nil {
		a.root.SetFocus(w)
	}
}

func (a *App) ShowKeyboardTester() {
	kt := ui.NewKeyTesterWidget()
	kt.Borders = a.borders
	kt.LookupBinding = func(combo string) string {
		for _, kb := range a.keybindings {
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
	if a.sidebar.Visible && a.sidebar.ActivePanel == "search" {
		a.search.ToggleReplaceMode()
	} else {
		a.search.SetReplaceMode(true)
		a.ShowPanel("search", a.search)
	}
	a.root.SetFocus(a.search)
}

func registerViewCommands(app *App) {
	reg := app.reg

	reg.Register(command.Command{
		ID: "sidebar.toggle", Title: "Toggle Sidebar",
		Handler: app.ToggleSidebar,
	})

	reg.Register(command.Command{
		ID: "sidebar.explorer", Title: "Show Explorer",
		Handler: func() {
			app.explorer.Reload()
			app.ShowPanel("explorer", app.explorer)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.search", Title: "Show Search",
		Handler: func() {
			app.ShowPanel("search", app.search)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.searchReplace", Title: "Search and Replace in Files",
		Handler: app.ToggleSearchReplace,
	})

	reg.Register(command.Command{
		ID: "sidebar.changes", Title: "Show Changes",
		Handler: func() {
			app.changes.Refresh()
			app.ShowPanel("changes", app.changes)
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.wider", Title: "Increase Sidebar Width",
		Handler: func() {
			if app.sidebar.Visible {
				app.SetSidebarWidth(app.splitPanel.DividerPos + 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.narrower", Title: "Decrease Sidebar Width",
		Handler: func() {
			if app.sidebar.Visible {
				app.SetSidebarWidth(app.splitPanel.DividerPos - 1)
			}
		},
	})

	reg.Register(command.Command{
		ID: "sidebar.focus", Title: "Focus Sidebar",
		Handler: app.FocusSidebar,
	})

	reg.Register(command.Command{
		ID: "panel.toggle", Title: "Toggle Panel",
		Handler: app.ToggleBottomPanel,
	})

	reg.Register(command.Command{
		ID: "panel.focus", Title: "Focus Panel",
		Handler: app.FocusPanel,
	})

	reg.Register(command.Command{
		ID: "terminal.new", Title: "New Terminal",
		Handler: app.SpawnTerminal,
	})

	reg.Register(command.Command{
		ID: "terminal.toggle", Title: "Toggle Terminal",
		Handler: app.ToggleTerminal,
	})

	reg.Register(command.Command{
		ID: "terminal.fullscreen", Title: "Toggle Terminal Fullscreen",
		Handler: app.ToggleTerminalFullscreen,
	})

	reg.Register(command.Command{
		ID: "terminal.closeAll", Title: "Close All Terminals",
		Handler: app.CloseAllTerminals,
	})

	reg.Register(command.Command{
		ID: "view.keyboardTester", Title: "Keyboard Tester",
		Handler: app.ShowKeyboardTester,
	})

	reg.Register(command.Command{
		ID: "about", Title: "About ttt",
		Handler: func() {
			openURL("https://github.com/eugenioenko/ttt")
		},
	})
}
