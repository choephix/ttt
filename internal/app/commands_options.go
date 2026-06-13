package app

import (
	"fmt"
	"strconv"

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

func (a *App) SetTabSizeOption(size int) {
	a.Settings.Editor.TabSize = size
	a.EditorGroup.TabSize = size
	a.EditorGroup.SetTabSize(size)
	config.SaveSettings(*a.Settings)
}

func (a *App) ShowTabSizePicker() {
	sizes := []int{2, 4, 8}
	var cmds []command.Command
	for _, s := range sizes {
		label := fmt.Sprintf("Tab Size: %d", s)
		cmds = append(cmds, command.Command{ID: strconv.Itoa(s), Title: label})
	}
	a.ShowPicker(cmds, func(id string) {
		if size, err := strconv.Atoi(id); err == nil {
			a.SetTabSizeOption(size)
		}
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

	items := []ui.ContextMenuItem{
		{Label: "Line Numbers", Command: "options.toggleLineNumbers", Checked: lineNumbersChecked},
		{Label: "Word Wrap", Command: "options.toggleWordWrap", Checked: wordWrapChecked},
		ui.MenuSep(),
		{Label: "Gutter Style", Command: "options.gutterStyle"},
		{Label: "Tab Size", Command: "options.tabSize"},
		ui.MenuSep(),
		{Label: "Switch Theme", Command: "theme.switch"},
		ui.MenuSep(),
		{Label: "Open Settings", Command: "settings.open"},
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
		ID: "options.gutterStyle", Title: "Change Gutter Style",
		Keywords: []string{"preferences", "settings", "editor", "view"},
		Handler:  app.ShowGutterStylePicker,
	})

	reg.Register(command.Command{
		ID: "options.tabSize", Title: "Change Tab Size",
		Keywords: []string{"preferences", "settings", "editor", "indentation"},
		Handler:  app.ShowTabSizePicker,
	})
}
