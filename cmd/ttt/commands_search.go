package main

import (
	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/ui"
)

func (a *App) OpenFind() {
	dv := a.editorGroup.ActiveDiffWidget()
	if dv != nil {
		a.showDiffFindBar(dv)
		return
	}
	findBar := ui.NewFindBarWidget()
	findBar.Borders = a.borders
	findBar.OnSearch = func(query string, opts ui.SearchOptions) []ui.FindMatch {
		matches, err := ui.FindInLines(a.editorGroup.Editor.Buf.Lines, query, opts)
		if err != nil {
			a.StatusWarn("Invalid regex: " + err.Error())
			return nil
		}
		a.editorGroup.SetSearch(query, matches)
		return matches
	}
	findBar.OnNavigate = func(match ui.FindMatch) {
		a.editorGroup.SetSearchActive(findBar.Current)
		a.editorGroup.Editor.Cursor.Line = match.Line
		a.editorGroup.Editor.Cursor.Col = match.Col
		a.editorGroup.ScrollToCursor()
	}
	findBar.OnDismiss = func() {
		a.DismissDialog()
		a.editorGroup.ClearSearch()
	}
	a.ShowFindBar(findBar)
}

func (a *App) OpenFindReplace() {
	bar := ui.NewReplaceBarWidget()
	bar.Borders = a.borders
	bar.OnSearch = func(query string, opts ui.SearchOptions) []ui.FindMatch {
		matches, err := ui.FindInLines(a.editorGroup.Editor.Buf.Lines, query, opts)
		if err != nil {
			a.StatusWarn("Invalid regex: " + err.Error())
			return nil
		}
		a.editorGroup.SetSearch(query, matches)
		return matches
	}
	bar.OnNavigate = func(match ui.FindMatch) {
		a.editorGroup.SetSearchActive(bar.Current)
		a.editorGroup.Editor.Cursor.Line = match.Line
		a.editorGroup.Editor.Cursor.Col = match.Col
		a.editorGroup.ScrollToCursor()
	}
	bar.OnReplace = func(match ui.FindMatch, replacement string) {
		a.editorGroup.ReplaceMatch(match, replacement)
	}
	bar.OnReplaceAll = func(query, replacement string) {
		a.editorGroup.ReplaceAll(query, replacement)
	}
	bar.OnDismiss = func() {
		a.DismissDialog()
		a.editorGroup.ClearSearch()
	}
	a.ShowDialog(bar)
}

func (a *App) ClearGlobalSearch() {
	a.search.Input.Text = ""
	a.search.Input.CursorPos = 0
	a.search.Groups = nil
	a.search.FlatList = nil
	a.search.Selected = 0
	a.search.ScrollTop = 0
	a.editorGroup.ClearSearch()
}

func registerSearchCommands(app *App) {
	reg := app.reg

	reg.Register(command.Command{
		ID: "search.find", Title: "Find",
		Handler: app.OpenFind,
	})

	reg.Register(command.Command{
		ID: "search.findNext", Title: "Find Next",
		Handler: func() { app.editorGroup.FindNext() },
	})

	reg.Register(command.Command{
		ID: "search.findPrev", Title: "Find Previous",
		Handler: func() { app.editorGroup.FindPrev() },
	})

	reg.Register(command.Command{
		ID: "search.clearFind", Title: "Clear Find Highlights",
		Handler: func() { app.editorGroup.ClearSearch() },
	})

	reg.Register(command.Command{
		ID: "search.replace", Title: "Find and Replace",
		Handler: app.OpenFindReplace,
	})

	reg.Register(command.Command{
		ID: "search.clear", Title: "Clear Search Results",
		Handler: app.ClearGlobalSearch,
	})
}
