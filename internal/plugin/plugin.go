package plugin

import (
	"log/slog"
	"path/filepath"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

type PluginCommand struct {
	ID      string
	Title   string
	Handler *lua.LFunction
}

type PluginKeybinding struct {
	Key     string
	Command string
}

type Plugin struct {
	Name     string
	Dir      string
	Repo     string
	Manifest Manifest
	Granted  PermissionSet
	Enabled  bool
	State    *lua.LState

	SidebarTitle       string
	SidebarMenuEntries []widgets.MenuEntry
	SidebarMenuFunc    *lua.LFunction
	RenderFunc         *lua.LFunction
	EventFunc          *lua.LFunction

	BottomTitle      string
	BottomRenderFunc *lua.LFunction
	BottomEventFunc  *lua.LFunction

	Commands          []PluginCommand
	PluginKeybindings []PluginKeybinding

	RequestRedraw   func()
	PostAsync       func(*PluginAsyncResult)
	Log             func(level, message string)
	ShowContextMenu func(entries []widgets.MenuEntry, x, y int, onCommand func(cmd string))
	ShowInfoDialog    func(title string, entries []widgets.KeyValueEntry)
	ShowConfirmDialog func(message string, onConfirm func())
	SimulateClick     func(x, y int)
	ScreenshotToFile  func(path string) error
	DebugDumpToFile   func(path string) error
	QuitApp           func()
	OpenDrawer        func(renderFunc *lua.LFunction, width, minWidth int)
	CloseDrawer       func()
	OpenTab           func(id string, renderFunc, eventFunc *lua.LFunction)
	CloseTab          func(id string)
	Borders         *term.BorderSet

	Editor     EditorAPI
	Filesystem FilesystemAPI
	System     SystemAPI
	Network    NetworkAPI

	EventListeners map[string][]*lua.LFunction

	LastError error
}

func (p *Plugin) Init() error {
	p.EventListeners = make(map[string][]*lua.LFunction)
	p.State = NewSandbox()
	setupTTTModule(p.State, p)
	setupEditorModule(p.State, p)
	setupFsModule(p.State, p)
	setupSystemModule(p.State, p)
	setupNetModule(p.State, p)
	setupEventsModule(p.State, p)

	entry := filepath.Join(p.Dir, p.Manifest.Entry)
	if err := p.State.DoFile(entry); err != nil {
		p.LastError = err
		p.State.Close()
		p.State = nil
		p.logError("init", err)
		return err
	}

	p.Enabled = true
	return nil
}

func (p *Plugin) InitFromSource(source string) error {
	p.EventListeners = make(map[string][]*lua.LFunction)
	p.State = NewSandbox()
	setupTTTModule(p.State, p)
	setupEditorModule(p.State, p)
	setupFsModule(p.State, p)
	setupSystemModule(p.State, p)
	setupNetModule(p.State, p)
	setupEventsModule(p.State, p)

	if err := p.State.DoString(source); err != nil {
		p.LastError = err
		p.State.Close()
		p.State = nil
		p.logError("init", err)
		return err
	}

	p.Enabled = true
	return nil
}

func (p *Plugin) logError(context string, err error) {
	slog.Error("plugin error", "plugin", p.Name, "context", context, "error", err)
	if p.Log != nil {
		p.Log("error", context+": "+err.Error())
	}
}

func (p *Plugin) Destroy() {
	if p.State != nil {
		p.State.Close()
		p.State = nil
	}
	p.Enabled = false
	p.RenderFunc = nil
	p.EventFunc = nil
	p.BottomRenderFunc = nil
	p.BottomEventFunc = nil
	p.Commands = nil
	p.PluginKeybindings = nil
	p.RequestRedraw = nil
	p.PostAsync = nil
	p.Log = nil
	p.ShowContextMenu = nil
	p.ShowConfirmDialog = nil
	p.OpenDrawer = nil
	p.CloseDrawer = nil
	p.OpenTab = nil
	p.CloseTab = nil
	p.EventListeners = nil
}

func (p *Plugin) CallSidebarAction(cmd string) {
	if p.SidebarMenuFunc != nil && p.State != nil {
		p.CallLuaFunc(p.SidebarMenuFunc, lua.LString(cmd))
	}
}

func (p *Plugin) DispatchEvent(name string, args ...lua.LValue) {
	listeners := p.EventListeners[name]
	if len(listeners) == 0 || p.State == nil {
		return
	}
	for _, fn := range listeners {
		if err := p.CallLuaFunc(fn, args...); err != nil {
			p.logError("event "+name, err)
		}
	}
}

func (p *Plugin) CallRender(proxy *PanelProxy) error {
	return p.CallRenderWith(p.RenderFunc, proxy)
}

func (p *Plugin) CallRenderWith(renderFunc *lua.LFunction, proxy *PanelProxy) error {
	if p.State == nil || renderFunc == nil {
		return nil
	}

	ud := PushPanelProxy(p.State, proxy)
	err := p.State.CallByParam(lua.P{
		Fn:      renderFunc,
		NRet:    0,
		Protect: true,
	}, ud)
	if err != nil {
		p.LastError = err
		p.logError("render", err)
	}
	return err
}
