package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/ui"
)

var explorerHelpEntries = []ui.InfoEntry{
	{Key: "Enter", Desc: "Open file or toggle folder"},
	{Key: "Space", Desc: "Open file or toggle folder"},
	{Key: "Shift+Enter", Desc: "Open context menu"},
	{Key: "Menu*", Desc: "Open context menu (terminal-dependent)"},
	{Key: "Left", Desc: "Collapse folder"},
	{Key: "Right", Desc: "Expand folder"},
	{Key: "Up / Down", Desc: "Navigate items"},
}

var searchHelpEntries = []ui.InfoEntry{
	{Key: "Enter", Desc: "Activate selected result"},
	{Key: "Up / Down", Desc: "Navigate results"},
	{Key: "Tab", Desc: "Next input field"},
	{Key: "Shift+Tab", Desc: "Previous input field"},
	{Key: "Alt+c", Desc: "Toggle case sensitivity"},
	{Key: "Alt+r", Desc: "Toggle regex mode"},
}

var changesHelpEntries = []ui.InfoEntry{
	{Key: "Space", Desc: "Toggle stage/unstage file"},
	{Key: "a", Desc: "Stage all files"},
	{Key: "u", Desc: "Unstage all files"},
	{Key: "d", Desc: "Discard selected file"},
	{Key: "D", Desc: "Discard all files in group"},
	{Key: "r", Desc: "Refresh changes"},
	{Key: "o / v", Desc: "Open file"},
	{Key: "c", Desc: "Open compact diff"},
	{Key: "e", Desc: "Open extended diff"},
	{Key: "Enter", Desc: "Open compact diff"},
	{Key: "Up / Down", Desc: "Navigate files"},
}

func (a *App) ShowPanelHelp(title string, entries []ui.InfoEntry) {
	dialog := ui.NewInfoDialogWidget(title, entries)
	dialog.Borders = a.Borders
	dialog.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(dialog)
}

func registerHelpCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID:       "explorer.help",
		Title:    "Explorer: Keyboard Shortcuts",
		Keywords: []string{"view", "help", "keybindings"},
		Handler: func() {
			app.ShowPanelHelp("Explorer Shortcuts", explorerHelpEntries)
		},
	})

	reg.Register(command.Command{
		ID:       "search.help",
		Title:    "Search: Keyboard Shortcuts",
		Keywords: []string{"search", "help", "keybindings"},
		Handler: func() {
			app.ShowPanelHelp("Search Shortcuts", searchHelpEntries)
		},
	})

	reg.Register(command.Command{
		ID:       "changes.help",
		Title:    "Changes: Keyboard Shortcuts",
		Keywords: []string{"git", "help", "keybindings"},
		Handler: func() {
			app.ShowPanelHelp("Changes Shortcuts", changesHelpEntries)
		},
	})
}
