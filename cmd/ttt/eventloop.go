package main

import (
	"fmt"
	"log/slog"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/render"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type blameResult struct {
	gen  int
 info *git.BlameInfo
}

func runEventLoop(
	screen *term.TcellScreen,
	renderer *render.Renderer,
	app *App,
	running *bool,
	quitPending *bool,
	closeTerminal func(panelID string),
) {
	lastBlameLine := -1
	lastBlameFile := ""
	lastBranchDir := app.workspace.Primary()
	blameGen := 0
	app.status.Branch = git.BranchName(lastBranchDir)
	app.status.TabSize = app.settings.TabSize

	syncStatus := func() {
		line, col := app.editorGroup.ActiveCursor()
		filePath := app.editorGroup.ActiveFilePath()
		app.status.FileName = filePath
		app.status.Line = line
		app.status.Col = col
		app.status.Dirty = app.editorGroup.IsDirty()
		app.status.CursorCount = app.editorGroup.MultiCursorCount()
		app.explorer.ActiveFile = filePath

		if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
			lang := app.editorGroup.Editor.Highlighter.Language()
			app.status.Language = lang
			_, _, lspOk := app.lspResolve(filePath, lang)
			app.status.LSP = lspOk
		} else {
			app.status.Language = ""
			app.status.LSP = false
		}
		if app.editorGroup.Editor != nil && app.editorGroup.Editor.TabSize > 0 {
			app.status.TabSize = app.editorGroup.Editor.TabSize
		} else {
			app.status.TabSize = app.settings.TabSize
		}

		repoDir := ""
		if filePath != "" && filePath != "untitled" {
			if folder := app.workspace.FolderForFile(filePath); folder != nil && folder.IsRepo {
				repoDir = folder.Path
			}
		}

		if repoDir != lastBranchDir {
			lastBranchDir = repoDir
			if repoDir != "" {
				app.status.Branch = git.BranchName(repoDir)
			} else {
				app.status.Branch = ""
			}
		}

		if filePath != lastBlameFile || line != lastBlameLine {
			lastBlameFile = filePath
			lastBlameLine = line
			app.status.Blame = ""
			if repoDir != "" {
				blameGen++
				gen := blameGen
				blameLine := line + 1
				go func() {
					info := git.BlameLine(repoDir, filePath, blameLine)
					screen.PostEvent(tcell.NewEventInterrupt(&blameResult{gen: gen, info: info}))
				}()
			}
		}
	}

	redraw := func() {
		cells := make([][]term.Cell, app.root.Height)
		for y := range cells {
			cells[y] = make([]term.Cell, app.root.Width)
		}
		app.root.Render(cells)
		resizeTerminals(app)
		renderer.SetCurrent(cells)
		if cx, cy, visible := app.root.CursorPosition(); visible {
			screen.ShowCursor(cx, cy)
		} else {
			screen.HideCursor()
		}
		renderer.Render(screen)
	}

	redraw()

	for *running {
		ev := screen.PollEvent()
		switch tev := ev.(type) {
		case *tcell.EventKey:
			if *quitPending && !(tev.Key() == tcell.KeyCtrlQ) {
				*quitPending = false
				app.status.DismissNotification()
			}
			if app.editorGroup.SignatureHelp != nil {
				switch tev.Key() {
				case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight,
					tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
					app.DismissSignatureHelp()
				}
			}
			slog.Debug("key", "key", tev.Key(), "rune", string(tev.Rune()), "mod", tev.Modifiers())
			app.root.HandleEvent(tev)
			app.RefreshAutocomplete()
			syncStatus()
			redraw()

		case *tcell.EventMouse:
			mx, my := tev.Position()
			btn := tev.Buttons()
			slog.Debug("mouse", "x", mx, "y", my, "btn", btn)
			app.DismissSignatureHelp()
			if btn == 0 {
				app.checkMouseHover(mx, my)
			} else {
				app.DismissHover()
			}
			app.root.HandleEvent(tev)
			syncStatus()
			redraw()

		case *tcell.EventResize:
			w, h := screen.Size()
			app.root.SetSize(w, h)
			resizeTerminals(app)
			renderer.Clear()
			redraw()

		case *tcell.EventInterrupt:
			switch v := tev.Data().(type) {
			case string:
				if v != "" {
					closeTerminal(v)
				}
			case *blameResult:
				if v.gen == blameGen && v.info != nil {
					app.status.Blame = fmt.Sprintf("%s, %s",
						v.info.Author, git.FormatRelativeTime(v.info.Time))
				}
			case *autocompleteTrigger:
				if !app.IsAutocompleteActive() {
					prefix := app.currentPrefix()
					if len(prefix) >= 1 {
						path := app.editorGroup.ActiveFilePath()
						lang := ""
						if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
							lang = app.editorGroup.Editor.Highlighter.Language()
						}
						line, col := app.editorGroup.ActiveCursor()
						app.RequestCompletions(path, lang, line, col)
					}
				}
			case *signatureHelpResult:
				if v.label != "" {
					app.ShowSignatureHelp(v)
				}
			case *completionResult:
				if len(v.items) > 0 {
					app.ShowAutocomplete(v.items, v.lspItems)
				}
			case *hoverResult:
				if v.text != "" && v.gen == app.hoverGen {
					app.ShowHover(v.text, v.anchorX, v.anchorY)
				}
			case *locationResult:
				if len(v.locations) > 0 {
					loc := v.locations[0]
					path := uriToPath(loc.URI)
					app.editorGroup.OpenFile(path)
					app.editorGroup.GoToLine(loc.Range.Start.Line + 1)
					app.root.SetFocus(app.editorGroup)
				}
			case *renameResult:
				if v.edit != nil && len(v.edit.Changes) > 0 {
					app.ApplyWorkspaceEdit(v.edit)
				}
			case *referencesResult:
				if len(v.locations) > 0 {
					app.ShowReferences(v.locations)
				} else {
					app.StatusNotify("No references found")
				}
			case *formattingResult:
				if len(v.edits) > 0 {
					app.ApplyTextEdits(v.edits)
				}
			case *diagnosticsResult:
				app.editorGroup.SetDiagnostics(v.path, v.diagnostics)
				if len(v.diagnostics) == 0 {
					delete(app.allDiagnostics, v.path)
				} else {
					app.allDiagnostics[v.path] = v.diagnostics
				}
				app.refreshProblems()
			}
			redraw()
		}
	}
}

func resizeTerminals(app *App) {
	if !app.contentSplit.ShowBottom {
		return
	}
	r := app.bottomPanel.GetRect()
	cols := r.W - terminalStripWidth
	rows := r.H - 2
	if cols <= 0 || rows <= 0 {
		return
	}
	for _, tt := range app.terminals {
		tt.term.Resize(cols, rows)
	}
}