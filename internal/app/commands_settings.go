package app

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/core/buffer"
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
			a.Renderer.Clear()
		}
	}
}

func (a *App) OpenDefaultSettings() {
	text := config.DefaultSettingsText()
	lines := strings.Split(text, "\n")
	// Remove trailing empty line from the split if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	buf := &buffer.Buffer{Lines: lines}
	a.EditorGroup.OpenBuffer("Default Settings (Read Only)", buf)
}

func registerSettingsCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "settings.reload", Title: "Reload Settings",
		Handler: app.ReloadSettings,
	})

	reg.Register(command.Command{
		ID: "options.defaultSettings", Title: "Preferences: Open Default Settings",
		Keywords: []string{"preferences", "settings", "configuration", "options", "defaults", "reference"},
		Handler:  app.OpenDefaultSettings,
	})
}
