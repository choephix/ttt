package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) ToggleLineNumbers() {
	a.Settings.Editor.LineNumbers = !a.Settings.Editor.LineNumbers
	a.EditorGroup.LineNumbers = a.Settings.Editor.LineNumbers
	if a.EditorGroup.Editor != nil {
		a.EditorGroup.Editor.LineNumbers = a.Settings.Editor.LineNumbers
	}
	config.SaveSettings(*a.Settings)
}

func (a *App) ToggleWordWrap() {
	a.Settings.Editor.WordWrap = !a.Settings.Editor.WordWrap
	config.SaveSettings(*a.Settings)
}

func (a *App) ToggleBracketPairColorization() {
	a.Settings.Editor.BracketPairColorization = !a.Settings.Editor.BracketPairColorization
	a.EditorGroup.BracketPairColorization = a.Settings.Editor.BracketPairColorization
	if a.EditorGroup.Editor != nil {
		a.EditorGroup.Editor.BracketPairColorization = a.Settings.Editor.BracketPairColorization
		a.EditorGroup.Editor.InvalidateBracketColors()
	}
	config.SaveSettings(*a.Settings)
}

func (a *App) ToggleGitGutter() {
	enabled := !a.Settings.Editor.IsGitGutterEnabled()
	a.Settings.Editor.GitGutter = &enabled
	config.SaveSettings(*a.Settings)
	if enabled {
		a.RequestGitGutterForActiveFile()
	} else if a.EditorGroup.Editor != nil {
		a.EditorGroup.Editor.LineChanges = nil
	}
}

func (a *App) SetGutterStyle(style string) {
	a.Settings.Editor.GutterStyle = style
	a.EditorGroup.GutterStyle = style
	if a.EditorGroup.Editor != nil {
		a.EditorGroup.Editor.GutterStyle = style
	}
	config.SaveSettings(*a.Settings)
}

func (a *App) ShowGutterStylePicker() {
	styles := []string{"minimal", "compact", "extended"}
	var cmds []command.Command
	for _, s := range styles {
		cmds = append(cmds, command.Command{ID: s, Title: s})
	}
	a.ShowPicker(cmds, func(id string) {
		a.SetGutterStyle(id)
	})
}

func (a *App) BuildOptionsMenu() []ui.ContextMenuItem {
	lineNumbersChecked := ui.MenuUnchecked
	if a.Settings.Editor.LineNumbers {
		lineNumbersChecked = ui.MenuChecked
	}

	wordWrapChecked := ui.MenuUnchecked
	if a.Settings.Editor.WordWrap {
		wordWrapChecked = ui.MenuChecked
	}

	bracketColorChecked := ui.MenuUnchecked
	if a.Settings.Editor.BracketPairColorization {
		bracketColorChecked = ui.MenuChecked
	}

	gitGutterChecked := ui.MenuUnchecked
	if a.Settings.Editor.IsGitGutterEnabled() {
		gitGutterChecked = ui.MenuChecked
	}

	items := []ui.ContextMenuItem{
		{Label: "Line Numbers", Command: "options.toggleLineNumbers", Checked: lineNumbersChecked},
		{Label: "Word Wrap", Command: "options.toggleWordWrap", Checked: wordWrapChecked},
		{Label: "Bracket Colors", Command: "options.toggleBracketColors", Checked: bracketColorChecked},
		{Label: "Git Gutter", Command: "options.toggleGitGutter", Checked: gitGutterChecked},
		ui.MenuSep(),
		{Label: "Gutter Style", Command: "options.gutterStyle"},
		{Label: "Indentation", Command: "options.indentation"},
		ui.MenuSep(),
		{Label: "Switch Theme", Command: "theme.switch"},
		ui.MenuSep(),
		{Label: "Open Settings", Command: "settings.open"},
		{Label: "Open Default Settings", Command: "options.defaultSettings"},
	}
	return items
}

func registerOptionsCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "options.toggleLineNumbers", Title: "Toggle Line Numbers",
		Keywords: []string{"preferences", "settings", "editor", "view"},
		Handler:  app.ToggleLineNumbers,
	})

	reg.Register(command.Command{
		ID: "options.toggleWordWrap", Title: "Toggle Word Wrap",
		Keywords: []string{"preferences", "settings", "editor", "view"},
		Handler:  app.ToggleWordWrap,
	})

	reg.Register(command.Command{
		ID: "options.toggleBracketColors", Title: "Toggle Bracket Pair Colorization",
		Handler: app.ToggleBracketPairColorization,
	})

	reg.Register(command.Command{
		ID: "options.toggleGitGutter", Title: "Toggle Git Gutter",
		Keywords: []string{"preferences", "settings", "editor", "view", "git"},
		Handler:  app.ToggleGitGutter,
	})

	reg.Register(command.Command{
		ID: "options.gutterStyle", Title: "Change Gutter Style",
		Keywords: []string{"preferences", "settings", "editor", "view"},
		Handler:  app.ShowGutterStylePicker,
	})

	reg.Register(command.Command{
		ID: "options.indentation", Title: "Editor Indentation",
		Keywords: []string{"preferences", "settings", "editor", "indentation", "tabs", "spaces"},
		Handler:  app.ShowIndentSettings,
	})
}
