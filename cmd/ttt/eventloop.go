package main

import (
	"fmt"
	"ttt/internal/command"
	"ttt/internal/git"
	"ttt/internal/render"
	"ttt/internal/term"
	"ttt/internal/ui"

	"github.com/gdamore/tcell/v2"
)

func runEventLoop(
	screen *term.TcellScreen,
	renderer *render.Renderer,
	cmdRegistry *command.Registry,
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
			app.root.HandleEvent(tev)
			syncStatus()
			redraw()

		case *tcell.EventMouse:
			mx, my := tev.Position()
			btn := tev.Buttons()

			if len(app.root.Overlays) > 0 {
				app.root.Overlays[len(app.root.Overlays)-1].Widget.HandleEvent(tev)
				redraw()
				continue
			}

			menuR := app.menuBar.GetRect()
			if btn&tcell.Button1 != 0 && my == menuR.Y {
				app.menuBar.HandleEvent(tev)
				redraw()
				continue
			}

			statusR := app.statusBar.GetRect()
			if btn&tcell.Button1 != 0 && my == statusR.Y {
				app.statusBar.HandleEvent(tev)
				redraw()
				continue
			}

			if btn&tcell.Button2 != 0 {
				handleRightClick(app, cmdRegistry, mx, my)
				redraw()
				continue
			}

			if app.splitPanel.HandleEvent(tev) == ui.EventConsumed {
				redraw()
				continue
			}

			if app.contentSplit.HandleEvent(tev) == ui.EventConsumed {
				redraw()
				continue
			}

			isWheel := btn&tcell.WheelUp != 0 || btn&tcell.WheelDown != 0

			if btn&tcell.Button1 != 0 || isWheel {
				panelRect := app.splitPanel.GetRect()
				inPanel := my >= panelRect.Y && my < panelRect.Y+panelRect.H &&
					mx >= panelRect.X && mx < panelRect.X+panelRect.W
				if inPanel {
					if app.sidebar.Visible {
						divX := app.splitPanel.DividerScreenX()
						if mx < divX {
							app.sidebar.HandleEvent(tev)
							if !isWheel {
								cmdRegistry.Execute("sidebar.focus")
							}
						} else {
							app.editorGroup.HandleEvent(tev)
							if !isWheel {
								cmdRegistry.Execute("editor.focus")
							}
						}
					} else {
						if btn&tcell.Button1 != 0 && mx == panelRect.X {
							cmdRegistry.Execute("sidebar.toggle")
						} else {
							app.editorGroup.HandleEvent(tev)
							if !isWheel {
								cmdRegistry.Execute("editor.focus")
							}
						}
					}
					redraw()
				}
			} else if btn == tcell.ButtonNone {
				if app.editorGroup.HandleEvent(tev) == ui.EventConsumed {
					redraw()
				}
			}

		case *tcell.EventResize:
			w, h := screen.Size()
			app.root.SetSize(w, h)
			renderer.Clear()
			redraw()
		}
	}
}
