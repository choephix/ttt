package app

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/term"
)

func (a *App) ReloadSettings() {
	s := config.LoadSettings()
	a.ApplySettings(s)
	a.StatusNotify("Settings reloaded")
}

func (a *App) ApplySettings(s config.Settings) {
	*a.Settings = s

	// Apply editor settings to the editor group and active editor
	a.EditorGroup.TabSize = s.Editor.TabSize
	a.EditorGroup.InsertSpaces = s.Editor.InsertSpaces
	a.EditorGroup.LineNumbers = s.Editor.LineNumbers
	a.EditorGroup.GutterStyle = s.Editor.GutterStyle
	a.EditorGroup.InsertFinalNewline = s.Editor.InsertFinalNewline
	a.EditorGroup.TrimTrailingWhitespace = s.Editor.TrimTrailingWhitespace

	a.EditorGroup.Editor.TabSize = s.Editor.TabSize
	a.EditorGroup.Editor.LineNumbers = s.Editor.LineNumbers
	a.EditorGroup.Editor.GutterStyle = s.Editor.GutterStyle
	a.EditorGroup.Editor.AutoIndent = s.Editor.IsAutoIndentEnabled()

	// Apply cursor style
	if a.Screen != nil {
		a.Screen.SetCursorStyle(term.ParseCursorStyle(s.Editor.CursorStyle))
	}

	// Apply search debounce
	a.Search.Debounce.DelayMs = s.Search.Debounce

	// Apply theme if changed
	if s.Theme != "" && a.Screen != nil {
		theme, err := config.LoadTheme(s.Theme)
		if err == nil {
			a.Screen.SetStyleMap(BuildStyleMap(theme))
			*a.Palette = BuildTerminalPalette(theme)
			*a.Borders = BuildBorderSet(theme.Borders)
			a.ApplyBorderStyle()
			a.Renderer.Clear()
		}
	}
}

func registerSettingsCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "settings.reload", Title: "Reload Settings",
		Handler: app.ReloadSettings,
	})
}
