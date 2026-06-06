package main

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/ui"
)

func registerCommands(app *App) {
	registerViewCommands(app)
	registerEditorCommands(app)
	registerSearchCommands(app)
	registerPaletteCommands(app)
	registerExplorerCommands(app)
	registerGitCommands(app)
	registerWorkspaceCommands(app)
	registerPRCommands(app)
	registerWidgetCallbacks(app)
	registerEscapeDismissers(app)
}

func bindKeys(root *ui.Root, reg *command.Registry, keybindings []config.KeyBinding) {
	for _, kb := range keybindings {
		if len(kb.Steps) == 0 {
			continue
		}
		cmdID := kb.Command
		reg.SetShortcut(cmdID, formatKeyBinding(kb.Key))
		if kb.IsChord() {
			steps := make([]ui.GlobalKeyBinding, len(kb.Steps))
			for i, step := range kb.Steps {
				key, mod, rn := comboToTcell(step)
				steps[i] = ui.GlobalKeyBinding{Key: key, Mod: mod, Rune: rn}
			}
			root.AddChordKey(steps, func() {
				reg.Execute(cmdID)
			})
		} else {
			key, mod, rn := comboToTcell(kb.Steps[0])
			handler := func() { reg.Execute(cmdID) }
			root.AddGlobalKey(key, mod, rn, handler)
			if config.ForceKeyCommands[cmdID] {
				root.AddForceKey(key, mod, rn, handler)
			}
		}
	}
}

func formatKeyBinding(key string) string {
	parts := strings.Fields(key)
	for i, part := range parts {
		tokens := strings.Split(part, "+")
		for j, t := range tokens {
			if t == "backtick" {
				tokens[j] = "`"
			} else if len(t) > 0 {
				tokens[j] = strings.ToUpper(t[:1]) + t[1:]
			}
		}
		parts[i] = strings.Join(tokens, "+")
	}
	return strings.Join(parts, " ")
}

func registerEscapeDismissers(app *App) {
	app.root.EscapeDismissers = []func() bool{
		func() bool {
			if app.IsAutocompleteActive() {
				app.DismissAutocomplete()
				return true
			}
			return false
		},
		func() bool {
			if app.editorGroup.SignatureHelp != nil {
				app.DismissSignatureHelp()
				return true
			}
			return false
		},
		func() bool {
			if app.editorGroup.Hover != nil {
				app.DismissHover()
				return true
			}
			return false
		},
		func() bool {
			if app.editorGroup.IsMultiCursorActive() {
				app.editorGroup.CollapseMultiCursor()
				return true
			}
			return false
		},
	}
	app.root.EscapeFallback = func() {
		app.FocusEditor()
	}
}
