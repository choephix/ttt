package main

import (
	"ttt/internal/command"
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
	syncStatus := func() {
		line, col := app.editorGroup.ActiveCursor()
		app.status.FileName = app.editorGroup.ActiveFilePath()
		app.status.Line = line
		app.status.Col = col
		app.status.Dirty = app.editorGroup.IsDirty()
	}

	redraw := func() {
		cells := make([][]term.Cell, app.root.Height)
		for y := range cells {
			cells[y] = make([]term.Cell, app.root.Width)
		}
		app.root.Render(cells)
		renderer.SetCurrent(cells)
		if app.editorGroup.IsEditorActive() {
			screen.ShowCursor(app.editorGroup.Editor.CursorX, app.editorGroup.Editor.CursorY)
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
			}

		case *tcell.EventResize:
			w, h := screen.Size()
			app.root.SetSize(w, h)
			renderer.Clear()
			redraw()
		}
	}
}
