package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/view"
	"github.com/eugenioenko/ttt/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

func registerDebugCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID:       "debug.simulateClick",
		Title:    "Debug: Simulate Click",
		Keywords: []string{"debug", "click", "mouse", "test"},
		Handler:  func() { app.debugSimulateClick() },
	})

	reg.Register(command.Command{
		ID:       "debug.runPlugin",
		Title:    "Debug: Run Current File as Plugin",
		Keywords: []string{"debug", "plugin", "lua", "test", "run"},
		Handler:  func() { app.debugRunPlugin() },
	})
}

func (a *App) debugSimulateClick() {
	submit := func(text string) {
		a.DismissDialog()
		parts := strings.Fields(text)
		if len(parts) < 2 {
			a.Status.SetNotification("Usage: x y", view.NotifyWarning, 3*time.Second)
			return
		}
		x, err1 := strconv.Atoi(parts[0])
		y, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			a.Status.SetNotification("Invalid coordinates", view.NotifyWarning, 3*time.Second)
			return
		}
		go func() {
			a.Screen.PostEvent(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone))
			time.Sleep(50 * time.Millisecond)
			a.Screen.PostEvent(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone))
		}()
	}

	input := widgets.NewInputWidget(widgets.InputConfig{
		Placeholder: "x y",
		OnSubmit:    submit,
	})

	dialog := widgets.NewDialogWidget(40)
	dialog.Title = "Simulate Click"
	dialog.Borders = *a.Borders
	dialog.Buttons = []widgets.DialogButton{
		{Label: "Click", Handler: func() { submit(input.Text()) }},
	}
	dialog.SetContent(input)
	a.ShowDialog(dialog)
}

func (a *App) debugRunPlugin() {
	if a.EditorGroup.Editor == nil {
		a.Status.SetNotification("No active editor", view.NotifyWarning, 3*time.Second)
		return
	}

	source := strings.Join(a.EditorGroup.Editor.Buf.Lines, "\n")
	if strings.TrimSpace(source) == "" {
		a.Status.SetNotification("Buffer is empty", view.NotifyWarning, 3*time.Second)
		return
	}

	path := a.EditorGroup.ActiveFilePath()
	name := "debug-plugin"
	if path != "" {
		parts := strings.Split(path, "/")
		name = strings.TrimSuffix(parts[len(parts)-1], ".lua")
	}

	p := &plugin.Plugin{
		Name:    name,
		Dir:     ".",
		Enabled: true,
		Manifest: plugin.Manifest{
			Name:  name,
			Entry: "inline",
		},
		Granted: plugin.PermissionSet{
			PanelSidebar: true,
			PanelBottom:  true,
			PanelDrawer:  true,
			PanelEditor:  true,
			Commands:     true,
			Keybindings:  true,
			EditorRead:   true,
			EditorWrite:  true,
			FsRead:       true,
			FsWrite:      true,
			SystemEnv:    true,
			NetworkHTTP:  true,
			EventsFile:   true,
			EventsEditor: true,
		},
	}

	if err := p.InitFromSource(source); err != nil {
		a.Status.SetNotification("Plugin error: "+err.Error(), view.NotifyError, 5*time.Second)
		return
	}

	a.PluginManager.RegisterDebugPlugin(p)
	a.wirePlugin(p)

	hasSidebar := p.SidebarTitle != ""
	hasBottom := p.BottomTitle != ""

	if hasSidebar {
		a.Sidebar.SetActivePanel("plugin." + p.Name)
	}
	if hasBottom {
		a.BottomPanel.SetActivePanel("plugin." + p.Name)
	}

	a.Status.SetNotification("Plugin loaded: "+name, view.NotifyInfo, 3*time.Second)
}
