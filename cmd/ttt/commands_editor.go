package main

import (
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) editorPathLang() (string, string) {
	path := a.editorGroup.ActiveFilePath()
	lang := ""
	if a.editorGroup.Editor != nil && a.editorGroup.Editor.Highlighter != nil {
		lang = a.editorGroup.Editor.Highlighter.Language()
	}
	return path, lang
}

func (a *App) withEditorLSP(action func(path, lang string, line, col int)) {
	path, lang := a.editorPathLang()
	line, col := a.editorGroup.ActiveCursor()
	action(path, lang, line, col)
}

func (a *App) TriggerAutocomplete() {
	if !a.settings.Autocomplete.Enabled {
		return
	}
	if a.IsAutocompleteActive() {
		a.DismissAutocomplete()
		return
	}
	path, lang := a.editorPathLang()
	if lang == "" {
		a.StatusWarn("No language detected for this file")
	} else if _, _, ok := a.lspResolve(path, lang); !ok {
		a.StatusWarn(lang + " language server is not configured. Add it to settings.json under lsp.servers")
	} else {
		line, col := a.editorGroup.ActiveCursor()
		a.RequestCompletions(path, lang, line, col)
	}
}

func (a *App) RenameSymbol() {
	if a.editorGroup.Editor == nil {
		return
	}
	path, lang := a.editorPathLang()
	line, col := a.editorGroup.ActiveCursor()
	word := a.wordAtCursor()
	a.ShowInputDialog("Rename", "New name", word, func(newName string) {
		if newName != "" && newName != word {
			a.RequestRename(path, lang, line, col, newName)
		}
	})
}

func (a *App) FormatSelection() {
	if a.editorGroup.Editor == nil {
		return
	}
	path, lang := a.editorPathLang()
	sel := a.editorGroup.Editor.Selection
	if sel == nil || !sel.Active {
		a.RequestFormatting(path, lang)
		return
	}
	start, end := sel.Range(a.editorGroup.Editor.Cursor.Line, a.editorGroup.Editor.Cursor.Col)
	a.RequestRangeFormatting(path, lang, start.Line, start.Col, end.Line, end.Col)
}

func (a *App) CloseTab() {
	if !a.editorGroup.IsDirty() {
		a.editorGroup.CloseTab()
		return
	}
	name := a.editorGroup.ActiveFileName()
	dialog := ui.NewConfirmDialogWidget3(
		"Save changes to "+name+"?",
		"Discard", "Cancel", "Save",
	)
	dialog.Borders = a.borders
	dialog.OnButton[0] = func() {
		a.DismissDialog()
		a.editorGroup.CloseTab()
	}
	dialog.OnButton[1] = func() {
		a.DismissDialog()
	}
	dialog.OnButton[2] = func() {
		a.DismissDialog()
		a.reg.Execute("file.save")
		a.editorGroup.CloseTab()
	}
	dialog.OnDismiss = func() {
		a.DismissDialog()
	}
	a.ShowDialog(dialog)
}

func (a *App) NewFile() {
	a.editorGroup.OpenBuffer("untitled", &buffer.Buffer{Lines: []string{""}})
	a.root.SetFocus(a.editorGroup)
}

func (a *App) SaveFileAs() {
	current := a.editorGroup.ActiveFilePath()
	initial := ""
	if current != "untitled" {
		initial = current
	}
	a.ShowInputDialog("Save As", "Filename", initial, func(path string) {
		if path != "" {
			a.editorGroup.SaveAs(path)
		}
	})
}

func (a *App) SaveFile() {
	path, lang := a.editorPathLang()
	if lang != "" {
		a.RunCodeActionsOnSave(path, lang)
		if a.settings.FormatOnSave {
			a.FormatOnSave(path, lang)
		}
	}
	if !a.editorGroup.Save() {
		a.SaveFileAs()
		return
	}
	path, lang = a.editorPathLang()
	if lang != "" {
		text := strings.Join(a.editorGroup.Editor.Buf.Lines, "\n")
		a.NotifyLSPSave(path, lang, text)
	}
}

func (a *App) Quit() {
	if !a.editorGroup.AnyDirty() || *a.quitPending {
		for _, tt := range a.terminals {
			tt.term.Close()
		}
		*a.running = false
		return
	}
	*a.quitPending = true
	a.StatusWarn("Unsaved changes. Press Ctrl+Q again to quit.")
}

func registerEditorCommands(app *App) {
	reg := app.reg

	reg.Register(command.Command{
		ID: "editor.focus", Title: "Focus Editor",
		Handler: app.FocusEditor,
	})

	reg.Register(command.Command{
		ID: "editor.autocomplete", Title: "Trigger Autocomplete",
		Handler: app.TriggerAutocomplete,
	})

	reg.Register(command.Command{
		ID: "editor.hover", Title: "Show Hover",
		Handler: func() {
			path, lang := app.editorPathLang()
			line, col := app.editorGroup.ActiveCursor()
			ax, ay, _ := app.editorGroup.CursorPosition()
			app.RequestHover(path, lang, line, col, ax, ay)
		},
	})

	reg.Register(command.Command{
		ID: "editor.goToDefinition", Title: "Go to Definition",
		Handler: func() { app.withEditorLSP(app.RequestDefinition) },
	})

	reg.Register(command.Command{
		ID: "editor.goToImplementation", Title: "Go to Implementation",
		Handler: func() { app.withEditorLSP(app.RequestImplementation) },
	})

	reg.Register(command.Command{
		ID: "editor.goToTypeDefinition", Title: "Go to Type Definition",
		Handler: func() { app.withEditorLSP(app.RequestTypeDefinition) },
	})

	reg.Register(command.Command{
		ID: "editor.findReferences", Title: "Find All References",
		Handler: func() { app.withEditorLSP(app.RequestReferences) },
	})

	reg.Register(command.Command{
		ID: "editor.rename", Title: "Rename Symbol",
		Handler: app.RenameSymbol,
	})

	reg.Register(command.Command{
		ID: "editor.organizeImports", Title: "Source Action: Organize Imports",
		Handler: func() {
			path, lang := app.editorPathLang()
			app.RequestCodeAction(path, lang, "source.organizeImports")
		},
	})

	reg.Register(command.Command{
		ID: "editor.fixAll", Title: "Source Action: Fix All",
		Handler: func() {
			path, lang := app.editorPathLang()
			app.RequestCodeAction(path, lang, "source.fixAll")
		},
	})

	reg.Register(command.Command{
		ID: "editor.formatDocument", Title: "Source Action: Format Document",
		Handler: func() {
			path, lang := app.editorPathLang()
			app.RequestFormatting(path, lang)
		},
	})

	reg.Register(command.Command{
		ID: "editor.formatSelection", Title: "Source Action: Format Selection",
		Handler: app.FormatSelection,
	})

	reg.Register(command.Command{
		ID: "tab.next", Title: "Next Tab",
		Handler: func() { app.editorGroup.NextTab() },
	})

	reg.Register(command.Command{
		ID: "tab.prev", Title: "Previous Tab",
		Handler: func() { app.editorGroup.PrevTab() },
	})

	reg.Register(command.Command{
		ID: "tab.close", Title: "Close Tab",
		Handler: app.CloseTab,
	})

	reg.Register(command.Command{
		ID: "tab.closeOthers", Title: "Close Other Tabs",
		Handler: func() { app.editorGroup.CloseOtherTabs() },
	})

	reg.Register(command.Command{
		ID: "tab.closeAll", Title: "Close All Tabs",
		Handler: func() { app.editorGroup.CloseAllTabs() },
	})

	reg.Register(command.Command{
		ID: "file.new", Title: "New File",
		Handler: app.NewFile,
	})

	reg.Register(command.Command{
		ID: "file.save", Title: "Save File",
		Handler: app.SaveFile,
	})

	reg.Register(command.Command{
		ID: "file.saveAs", Title: "Save As...",
		Handler: app.SaveFileAs,
	})

	reg.Register(command.Command{
		ID: "editor.undo", Title: "Undo",
		Handler: func() { app.editorGroup.Undo() },
	})

	reg.Register(command.Command{
		ID: "editor.redo", Title: "Redo",
		Handler: func() { app.editorGroup.Redo() },
	})

	reg.Register(command.Command{
		ID: "editor.selectAll", Title: "Select All",
		Handler: func() { app.editorGroup.SelectAll() },
	})

	reg.Register(command.Command{
		ID: "editor.copy", Title: "Copy",
		Handler: func() { app.editorGroup.Copy() },
	})

	reg.Register(command.Command{
		ID: "editor.cut", Title: "Cut",
		Handler: func() { app.editorGroup.Cut() },
	})

	reg.Register(command.Command{
		ID: "editor.paste", Title: "Paste",
		Handler: func() { app.editorGroup.Paste() },
	})

	reg.Register(command.Command{
		ID: "editor.moveLineUp", Title: "Move Line Up",
		Handler: func() { app.editorGroup.MoveLineUp() },
	})
	reg.Register(command.Command{
		ID: "editor.moveLineDown", Title: "Move Line Down",
		Handler: func() { app.editorGroup.MoveLineDown() },
	})
	reg.Register(command.Command{
		ID: "editor.duplicateLine", Title: "Duplicate Line",
		Handler: func() { app.editorGroup.DuplicateLine() },
	})
	reg.Register(command.Command{
		ID: "editor.deleteLine", Title: "Delete Line",
		Handler: func() { app.editorGroup.DeleteLine() },
	})
	reg.Register(command.Command{
		ID: "editor.insertLineBelow", Title: "Insert Line Below",
		Handler: func() { app.editorGroup.InsertLineBelow() },
	})
	reg.Register(command.Command{
		ID: "editor.insertLineAbove", Title: "Insert Line Above",
		Handler: func() { app.editorGroup.InsertLineAbove() },
	})
	reg.Register(command.Command{
		ID: "editor.toggleComment", Title: "Toggle Line Comment",
		Handler: func() { app.editorGroup.ToggleLineComment() },
	})
	reg.Register(command.Command{
		ID: "editor.moveWordLeft", Title: "Move Word Left",
		Handler: func() { app.editorGroup.MoveWordLeft(false) },
	})
	reg.Register(command.Command{
		ID: "editor.moveWordRight", Title: "Move Word Right",
		Handler: func() { app.editorGroup.MoveWordRight(false) },
	})
	reg.Register(command.Command{
		ID: "editor.selectWordLeft", Title: "Select Word Left",
		Handler: func() { app.editorGroup.MoveWordLeft(true) },
	})
	reg.Register(command.Command{
		ID: "editor.selectWordRight", Title: "Select Word Right",
		Handler: func() { app.editorGroup.MoveWordRight(true) },
	})
	reg.Register(command.Command{
		ID: "editor.deleteWordLeft", Title: "Delete Word Left",
		Handler: func() { app.editorGroup.DeleteWordLeft() },
	})
	reg.Register(command.Command{
		ID: "editor.deleteWordRight", Title: "Delete Word Right",
		Handler: func() { app.editorGroup.DeleteWordRight() },
	})

	reg.Register(command.Command{
		ID: "multicursor.selectNext", Title: "Add Next Occurrence",
		Handler: func() { app.editorGroup.SelectNextOccurrence() },
	})
	reg.Register(command.Command{
		ID: "multicursor.selectAll", Title: "Select All Occurrences",
		Handler: func() { app.editorGroup.SelectAllOccurrences() },
	})
	reg.Register(command.Command{
		ID: "multicursor.undoCursor", Title: "Undo Last Cursor",
		Handler: func() { app.editorGroup.UndoLastCursor() },
	})

	reg.Register(command.Command{
		ID: "editor.quit", Title: "Quit",
		Handler: app.Quit,
	})
}
