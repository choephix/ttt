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

// ApplySettings is the single live-apply path: anything that can take effect
// without a restart belongs here, so every caller produces identical results.
func (a *App) ApplySettings(s config.Settings) {
	*a.Settings = s

	// Apply editor settings to the editor group and active editor
	a.EditorGroup.TabSize = s.Editor.TabSize
	a.EditorGroup.InsertSpaces = s.Editor.InsertSpaces
	a.EditorGroup.LineNumbers = s.Editor.LineNumbers
	a.EditorGroup.GutterStyle = s.Editor.GutterStyle
	a.EditorGroup.InsertFinalNewline = s.Editor.InsertFinalNewline
	a.EditorGroup.TrimTrailingWhitespace = s.Editor.TrimTrailingWhitespace
	a.EditorGroup.WordWrap = s.Editor.WordWrap
	a.EditorGroup.BracketPairColorization = s.Editor.BracketPairColorization

	if a.EditorGroup.Editor != nil {
		a.EditorGroup.Editor.TabSize = s.Editor.TabSize
		a.EditorGroup.Editor.LineNumbers = s.Editor.LineNumbers
		a.EditorGroup.Editor.GutterStyle = s.Editor.GutterStyle
		a.EditorGroup.Editor.AutoDedent = s.Editor.IsAutoDedentEnabled()
		a.EditorGroup.Editor.WordWrap = s.Editor.WordWrap
		a.EditorGroup.Editor.BracketPairColorization = s.Editor.BracketPairColorization
		a.EditorGroup.Editor.InvalidateBracketColors()
	}

	// Apply cursor style
	if a.Screen != nil {
		a.Screen.SetCursorStyle(term.ParseCursorStyle(s.Editor.CursorStyle))
	}

	// Apply search debounce
	a.Search.Debounce.DelayMs = s.Search.Debounce

	if s.Editor.IsGitGutterEnabled() {
		a.RequestGitGutterForActiveFile()
	} else if a.EditorGroup.Editor != nil {
		a.EditorGroup.Editor.LineChanges = nil
	}

	if a.Explorer != nil && a.Explorer.Settings != s.Explorer {
		a.Explorer.Settings = s.Explorer
		a.Explorer.Reload()
	}

	// An empty theme name means the built-in default, and must still be applied —
	// otherwise switching back to it leaves the previous theme's colors on screen.
	var themeBorders *term.BorderSet
	if a.Screen != nil {
		theme, ok := config.DefaultTheme(), s.Theme == ""
		if !ok {
			loaded, err := config.LoadTheme(s.Theme)
			theme, ok = loaded, err == nil
		}
		if ok {
			a.Screen.SetStyleMap(BuildStyleMap(theme))
			*a.Palette = BuildTerminalPalette(theme)
			borders := BuildBorderSet(theme.Borders)
			*a.Borders = borders
			themeBorders = &borders
			a.Renderer.Clear()
		}
	}

	// Overrides what the theme resolved, so it must run last and unconditionally.
	// Passing the borders just built avoids reloading the theme from disk.
	a.applyBorderStyle(themeBorders)
}

func registerSettingsCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "settings.reload", Title: "Reload Settings",
		Handler: app.ReloadSettings,
	})
}
