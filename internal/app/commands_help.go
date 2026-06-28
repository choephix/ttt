package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

var explorerHelpEntries = []widgets.KeyValueEntry{
	{Key: "Click", Value: "Open file in preview"},
	{Key: "Double-click", Value: "Open file permanently"},
	{Key: "Enter", Value: "Open file or toggle folder"},
	{Key: "Space", Value: "Open file or toggle folder"},
	{Key: "Shift+Enter", Value: "Open context menu"},
	{Key: "Menu*", Value: "Open context menu (terminal-dependent)"},
	{Key: "Left", Value: "Collapse folder"},
	{Key: "Right", Value: "Expand folder"},
	{Key: "Up / Down", Value: "Navigate items"},
}

var searchHelpEntries = []widgets.KeyValueEntry{
	{Key: "Enter", Value: "Activate selected result"},
	{Key: "Up / Down", Value: "Navigate results"},
	{Key: "Tab", Value: "Next input field"},
	{Key: "Shift+Tab", Value: "Previous input field"},
	{Key: "Alt+c", Value: "Toggle case sensitivity"},
	{Key: "Alt+r", Value: "Toggle regex mode"},
}

var changesHelpEntries = []widgets.KeyValueEntry{
	{Key: "Space / s", Value: "Toggle stage/unstage file"},
	{Key: "a", Value: "Stage all files"},
	{Key: "u", Value: "Unstage all files"},
	{Key: "d", Value: "Discard selected file"},
	{Key: "D", Value: "Discard all files in group"},
	{Key: "r", Value: "Refresh changes"},
	{Key: "o / v", Value: "Open file"},
	{Key: "c", Value: "Open compact diff"},
	{Key: "e", Value: "Open extended diff"},
	{Key: "Enter", Value: "Open compact diff"},
	{Key: "Up / Down", Value: "Navigate files"},
}

func (a *App) ShowPanelHelp(title string, entries []widgets.KeyValueEntry, description ...string) {
	if len(description) == 0 {
		a.ShowInfoDialog(title, entries)
		return
	}
	kv := widgets.NewKeyValueListWidget(entries)
	para := widgets.NewParagraphWidget(description[0])
	para.Box.MarginTop = 1
	vstack := widgets.NewVStackWidget(kv, para)

	dialog := widgets.NewDialogWidget(60)
	dialog.Title = title
	dialog.Borders = *a.Borders
	dialog.SetContent(vstack)
	dialog.Buttons = []widgets.DialogButton{
		{Label: "&Close", Handler: func() { a.DismissDialog() }},
	}
	dialog.OnDismiss = func() { a.DismissDialog() }
	dialog.Build()

	adapter := ui.NewWidgetAdapter(dialog)
	a.ShowDialog(adapter)
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
			app.ShowPanelHelp("Changes Shortcuts", changesHelpEntries,
				"Type a commit message in the input above and press Enter to commit. The commit applies to the currently selected group.")
		},
	})
}
