package main

import (
	"fmt"
	"strconv"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) OpenCommandPalette(fileMode bool, initialText ...string) {
	palette := ui.NewCommandPaletteWidget(a.reg.List())
	palette.Borders = a.borders
	palette.SetFiles(a.workspace.Paths())
	if len(initialText) > 0 {
		palette.Input.SetText(initialText[0])
	} else if fileMode {
		palette.Input.SetText("")
	}
	palette.OnExecute = func(id string) {
		a.DismissDialog()
		a.reg.Execute(id)
	}
	palette.OnOpenFile = func(absPath string) {
		a.DismissDialog()
		a.editorGroup.OpenFile(absPath)
	}
	palette.OnGoToLine = func(line int) {
		a.DismissDialog()
		a.editorGroup.GoToLine(line)
	}
	palette.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(palette)
}

func (a *App) ShowThemePicker() {
	names := config.ListThemes()
	if len(names) == 0 {
		return
	}
	var cmds []command.Command
	for _, name := range names {
		cmds = append(cmds, command.Command{ID: name, Title: name})
	}
	picker := ui.NewCommandPaletteWidget(cmds)
	picker.Borders = a.borders
	originalStyleMap := a.screen.GetStyleMap()
	originalPalette := *a.palette
	applyTheme := func(theme config.ThemeConfig) {
		a.screen.SetStyleMap(buildStyleMap(theme))
		*a.palette = buildTerminalPalette(theme)
		a.renderer.Clear()
	}
	picker.OnSelectionChange = func(name string) {
		theme, err := config.LoadTheme(name)
		if err != nil {
			return
		}
		applyTheme(theme)
	}
	picker.OnExecute = func(name string) {
		a.DismissDialog()
		theme, err := config.LoadTheme(name)
		if err != nil {
			return
		}
		applyTheme(theme)
		a.settings.Theme = name
		config.SaveSettings(*a.settings)
	}
	picker.OnDismiss = func() {
		a.DismissDialog()
		a.screen.SetStyleMap(originalStyleMap)
		*a.palette = originalPalette
		a.renderer.Clear()
	}
	a.ShowDialog(picker)
}

func (a *App) ShowIndentPicker() {
	var cmds []command.Command
	sizes := []int{1, 2, 3, 4, 6, 8}
	for _, s := range sizes {
		label := fmt.Sprintf("Spaces: %d", s)
		cmds = append(cmds, command.Command{ID: strconv.Itoa(s), Title: label})
	}
	cmds = append(cmds, command.Command{ID: "tabs", Title: "Indent Using Tabs"})
	cmds = append(cmds, command.Command{ID: "detect", Title: "Detect from Content"})
	a.ShowPicker(cmds, func(id string) {
		if id == "detect" {
			if a.editorGroup.Editor != nil && a.editorGroup.Editor.Buf != nil {
				if info := buffer.DetectIndent(a.editorGroup.Editor.Buf.Lines); info.Size > 0 {
					a.editorGroup.SetTabSize(info.Size)
				}
			}
		} else if id == "tabs" {
			a.editorGroup.SetTabSize(4)
		} else if size, err := strconv.Atoi(id); err == nil {
			a.editorGroup.SetTabSize(size)
		}
	})
}

func registerPaletteCommands(app *App) {
	reg := app.reg

	reg.Register(command.Command{
		ID: "command.palette", Title: "Command Palette",
		Handler: func() { app.OpenCommandPalette(false) },
	})

	reg.Register(command.Command{
		ID: "file.quickOpen", Title: "Go to File",
		Handler: func() { app.OpenCommandPalette(true) },
	})

	reg.Register(command.Command{
		ID: "editor.goToLine", Title: "Go to Line",
		Handler: func() { app.OpenCommandPalette(false, ":") },
	})

	reg.Register(command.Command{
		ID: "theme.switch", Title: "Switch Theme",
		Handler: app.ShowThemePicker,
	})

	app.statusBar.OnIndentClick = app.ShowIndentPicker

	reg.Register(command.Command{
		ID: "editor.indentation", Title: "Change Indentation",
		Handler: app.ShowIndentPicker,
	})

	reg.Register(command.Command{
		ID: "settings.open", Title: "Preferences: Open Settings",
		Handler: func() {
			path := config.ConfigFilePath("settings.json")
			app.editorGroup.OpenFile(path)
		},
	})

	reg.Register(command.Command{
		ID: "keybindings.open", Title: "Preferences: Open Keyboard Shortcuts",
		Handler: func() {
			path := config.ConfigFilePath("keybindings.json")
			app.editorGroup.OpenFile(path)
		},
	})
}
