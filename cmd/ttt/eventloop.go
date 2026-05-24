package main

import (
	"fmt"
	"log/slog"
	"ttt/internal/git"
	"ttt/internal/render"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

func runEventLoop(
	screen *term.TcellScreen,
	renderer *render.Renderer,
	app *appWidgets,
	running *bool,
	quitPending *bool,
) {
	lastBlameLine := -1
	lastBlameFile := ""
	app.status.Branch = git.BranchName(app.cwd)
	app.status.TabSize = app.settings.TabSize

	syncStatus := func() {
		line, col := app.editorGroup.ActiveCursor()
		filePath := app.editorGroup.ActiveFilePath()
		app.status.FileName = filePath
		app.status.Line = line
		app.status.Col = col
		app.status.Dirty = app.editorGroup.IsDirty()
		app.explorer.ActiveFile = filePath

		if app.editorGroup.Editor != nil && app.editorGroup.Editor.Highlighter != nil {
			app.status.Language = app.editorGroup.Editor.Highlighter.Language()
		} else {
			app.status.Language = ""
		}
		if app.editorGroup.Editor != nil && app.editorGroup.Editor.TabSize > 0 {
			app.status.TabSize = app.editorGroup.Editor.TabSize
		} else {
			app.status.TabSize = app.settings.TabSize
		}

		if filePath != lastBlameFile || line != lastBlameLine {
			lastBlameFile = filePath
			lastBlameLine = line
			app.status.Blame = ""
			if filePath != "" {
				info := git.BlameLine(app.cwd, filePath, line+1)
				if info != nil {
					app.status.Blame = fmt.Sprintf("%s, %s",
						info.Author, git.FormatRelativeTime(info.Time))
				}
			}
		}
	}

	redraw := func() {
		cells := make([][]term.Cell, app.root.Height)
		for y := range cells {
			cells[y] = make([]term.Cell, app.root.Width)
		}
		app.root.Render(cells)
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
				app.status.Message = ""
			}
			slog.Debug("key", "key", tev.Key(), "rune", string(tev.Rune()), "mod", tev.Modifiers())
			app.root.HandleEvent(tev)
			syncStatus()
			redraw()

		case *tcell.EventMouse:
			mx, my := tev.Position()
			btn := tev.Buttons()
			slog.Debug("mouse", "x", mx, "y", my, "btn", btn)
			app.root.HandleEvent(tev)
			syncStatus()
			redraw()

		case *tcell.EventResize:
			w, h := screen.Size()
			app.root.SetSize(w, h)
			renderer.Clear()
			redraw()
		}
	}
}
