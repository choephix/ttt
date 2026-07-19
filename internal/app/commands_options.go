package app

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

func (a *App) SaveAndApplySettings() {
	config.SaveSettings(*a.Settings)
	a.ApplySettings(*a.Settings)
}

func (a *App) ToggleLineNumbers() {
	a.Settings.Editor.LineNumbers = !a.Settings.Editor.LineNumbers
	a.SaveAndApplySettings()
}

func (a *App) ToggleWordWrap() {
	a.Settings.Editor.WordWrap = !a.Settings.Editor.WordWrap
	a.SaveAndApplySettings()
}

func (a *App) ToggleAutoDedent() {
	enabled := !a.Settings.Editor.IsAutoDedentEnabled()
	a.Settings.Editor.AutoDedent = &enabled
	a.SaveAndApplySettings()
}

func (a *App) ToggleSyntaxHighlight() {
	enabled := !a.Settings.Editor.IsSyntaxHighlightEnabled()
	a.Settings.Editor.SyntaxHighlight = &enabled
	a.SaveAndApplySettings()
	a.StatusNotify("Restart to apply syntax highlight changes")
}

func (a *App) ToggleBracketPairColorization() {
	a.Settings.Editor.BracketPairColorization = !a.Settings.Editor.BracketPairColorization
	a.SaveAndApplySettings()
}

func (a *App) ToggleLSP() {
	enabled := !a.Settings.LSP.IsEnabled()
	a.Settings.LSP.Enabled = &enabled
	a.SaveAndApplySettings()
}

func (a *App) ToggleGitGutter() {
	enabled := !a.Settings.Editor.IsGitGutterEnabled()
	a.Settings.Editor.GitGutter = &enabled
	a.SaveAndApplySettings()
}

func (a *App) SetGutterStyle(style string) {
	a.Settings.Editor.GutterStyle = style
	a.SaveAndApplySettings()
}

var styleLabels = map[string]string{
	"ascii": "ASCII",
}

func styleLabel(id string) string {
	if label, ok := styleLabels[id]; ok {
		return label
	}
	if id == "" {
		return ""
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

func GutterStyleItems() []widgets.SelectItem {
	items := make([]widgets.SelectItem, 0, len(config.GutterStyles))
	for _, id := range config.GutterStyles {
		items = append(items, widgets.SelectItem{ID: id, Label: styleLabel(id)})
	}
	return items
}

func BorderStyleItems() []widgets.SelectItem {
	items := make([]widgets.SelectItem, 0, len(config.BorderStyles))
	for _, id := range config.BorderStyles {
		// "theme" is accepted in settings.json but behaves identically to
		// "default", so it is not offered as a separate choice.
		if id == "theme" {
			continue
		}
		items = append(items, widgets.SelectItem{ID: id, Label: styleLabel(id)})
	}
	return items
}

func (a *App) ShowGutterStylePicker() {
	a.ShowSelectDialog("Gutter Style", GutterStyleItems(), func(id string) {
		a.SetGutterStyle(id)
	}, nil)
}

func (a *App) SetBorderStyle(style string) {
	a.Settings.Editor.BorderStyle = style
	a.SaveAndApplySettings()
}

func (a *App) ApplyBorderStyle() { a.applyBorderStyle(nil) }

// themeBorders, when non-nil, is the border set already resolved from the
// current theme, so the theme need not be reloaded from disk.
func (a *App) applyBorderStyle(themeBorders *term.BorderSet) {
	style := a.Settings.Editor.BorderStyle
	switch style {
	case "default", "theme", "":
		// Fall back to the theme's border set. Rebuilding it here (rather than
		// relying on whatever is currently in a.Borders) is what makes switching
		// from an explicit style back to "default" actually take effect.
		if themeBorders != nil {
			*a.Borders = *themeBorders
		} else if a.Settings.Theme != "" {
			if theme, err := config.LoadTheme(a.Settings.Theme); err == nil {
				*a.Borders = BuildBorderSet(theme.Borders)
			}
		}
	case "rounded":
		*a.Borders = term.RoundedBorderSet()
	case "sharp":
		*a.Borders = term.SingleBorderSet()
	case "double":
		*a.Borders = term.DoubleBorderSet()
	case "bold":
		*a.Borders = term.BoldBorderSet()
	case "ascii":
		*a.Borders = term.AsciiBorderSet()
	case "none":
		*a.Borders = term.NoneBorderSet()
	}
}

func (a *App) ShowBorderStylePicker() {
	a.ShowSelectDialog("Border Style", BorderStyleItems(), func(id string) {
		a.SetBorderStyle(id)
	}, nil)
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

	autoDedentChecked := ui.MenuUnchecked
	if a.Settings.Editor.IsAutoDedentEnabled() {
		autoDedentChecked = ui.MenuChecked
	}

	lspChecked := ui.MenuUnchecked
	if a.Settings.LSP.IsEnabled() {
		lspChecked = ui.MenuChecked
	}

	gitGutterChecked := ui.MenuUnchecked
	if a.Settings.Editor.IsGitGutterEnabled() {
		gitGutterChecked = ui.MenuChecked
	}

	syntaxChecked := ui.MenuUnchecked
	if a.Settings.Editor.IsSyntaxHighlightEnabled() {
		syntaxChecked = ui.MenuChecked
	}

	items := []ui.ContextMenuItem{
		{Label: "Line Numbers", Command: "options.toggleLineNumbers", Checked: lineNumbersChecked},
		{Label: "Word Wrap", Command: "options.toggleWordWrap", Checked: wordWrapChecked},
		{Label: "Auto Dedent", Command: "options.toggleAutoDedent", Checked: autoDedentChecked},
		{Label: "Syntax Highlight", Command: "options.toggleSyntaxHighlight", Checked: syntaxChecked},
		{Label: "Bracket Colors", Command: "options.toggleBracketColors", Checked: bracketColorChecked},
		{Label: "LSP Code Assist", Command: "options.toggleLSP", Checked: lspChecked},
		{Label: "Git Gutter", Command: "options.toggleGitGutter", Checked: gitGutterChecked},
		ui.MenuSep(),
		{Label: "Gutter Style", Command: "options.gutterStyle"},
		{Label: "Border Style", Command: "options.borderStyle"},
		{Label: "Indentation", Command: "options.indentation"},
		ui.MenuSep(),
		{Label: "Switch Theme", Command: "theme.switch"},
		ui.MenuSep(),
		{Label: "Open Settings", Command: "settings.openUI"},
	}
	return items
}

func registerOptionsCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "options.toggleSyntaxHighlight", Title: "Toggle Syntax Highlight",
		Keywords: []string{"preferences", "settings", "editor", "view", "performance"},
		Handler:  app.ToggleSyntaxHighlight,
	})

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
		ID: "options.toggleAutoDedent", Title: "Toggle Auto Dedent",
		Keywords: []string{"preferences", "settings", "editor", "indentation", "dedent", "bracket"},
		Handler:  app.ToggleAutoDedent,
	})

	reg.Register(command.Command{
		ID: "options.toggleBracketColors", Title: "Toggle Bracket Pair Colorization",
		Handler: app.ToggleBracketPairColorization,
	})

	reg.Register(command.Command{
		ID: "options.toggleLSP", Title: "Toggle LSP",
		Keywords: []string{"preferences", "settings", "language", "server", "autocomplete"},
		Handler:  app.ToggleLSP,
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
		ID: "options.borderStyle", Title: "Change Border Style",
		Keywords: []string{"preferences", "settings", "editor", "view", "borders", "rounded", "sharp"},
		Handler:  app.ShowBorderStylePicker,
	})

	reg.Register(command.Command{
		ID: "options.indentation", Title: "Editor Indentation",
		Keywords: []string{"preferences", "settings", "editor", "indentation", "tabs", "spaces"},
		Handler:  app.ShowIndentSettings,
	})
}
