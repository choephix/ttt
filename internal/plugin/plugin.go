package plugin

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

type MarkdownSpan struct {
	Text  string
	Style term.Style
}

type MarkdownLine struct {
	Spans []MarkdownSpan
}

type PluginCommand struct {
	ID      string
	Title   string
	Handler func() error
}

type PluginKeybinding struct {
	Key     string
	Command string
}

type HostCallbacks struct {
	RequestRedraw   func()
	PostAsync       func(*PluginAsyncResult)
	Log             func(level, message string)
	ShowContextMenu func(entries []widgets.MenuEntry, x, y int, onCommand func(cmd string))
	ShowInfoDialog  func(title string, entries []widgets.KeyValueEntry)
	ShowConfirmDialog func(message string, onConfirm func())
	OpenDrawer      func(panel *PluginPanelWidget, width, minWidth int)
	CloseDrawer     func()
	OpenTab         func(id string, panel *PluginPanelWidget)
	CloseTab        func(id string)
	RenderMarkdown  func(text string) []MarkdownLine
	Borders         *term.BorderSet
}

type DebugCallbacks struct {
	SimulateClick    func(x, y int)
	SimulateDrag     func(x1, y1, x2, y2 int)
	ScreenshotToFile func(path string) error
	DebugDumpToFile  func(path string) error
	QuitApp          func()
}

type Plugin struct {
	mu sync.Mutex

	Name     string
	Dir      string
	Repo     string
	Manifest Manifest
	Granted  PermissionSet
	Enabled  bool
	State    *lua.LState

	SidebarTitle       string
	SidebarMenuEntries []widgets.MenuEntry
	sidebarMenuFunc    *lua.LFunction
	RenderFunc         *lua.LFunction
	EventFunc          *lua.LFunction

	BottomTitle      string
	BottomRenderFunc *lua.LFunction
	BottomEventFunc  *lua.LFunction

	Commands          []PluginCommand
	PluginKeybindings []PluginKeybinding

	Host  HostCallbacks
	Debug DebugCallbacks

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
	absEntry, err := filepath.Abs(entry)
	if err != nil {
		p.State.Close()
		p.State = nil
		return fmt.Errorf("invalid entry path: %w", err)
	}
	absDir, _ := filepath.Abs(p.Dir)
	if !strings.HasPrefix(absEntry, absDir+string(filepath.Separator)) {
		p.State.Close()
		p.State = nil
		return fmt.Errorf("entry path %q escapes plugin directory", p.Manifest.Entry)
	}
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
	if p.Host.Log != nil {
		p.Host.Log("error", context+": "+err.Error())
	}
}

func (p *Plugin) SafePostAsync(result *PluginAsyncResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Host.PostAsync != nil {
		p.Host.PostAsync(result)
	}
}

func (p *Plugin) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.State != nil {
		p.State.Close()
		p.State = nil
	}
	p.Enabled = false
	p.RenderFunc = nil
	p.EventFunc = nil
	p.BottomRenderFunc = nil
	p.BottomEventFunc = nil
	p.sidebarMenuFunc = nil
	p.Commands = nil
	p.PluginKeybindings = nil
	p.Host = HostCallbacks{}
	p.Debug = DebugCallbacks{}
	p.EventListeners = nil
}

func (p *Plugin) HasSidebarMenu() bool {
	return p.sidebarMenuFunc != nil
}

func (p *Plugin) CallSidebarAction(cmd string) {
	if p.sidebarMenuFunc != nil && p.State != nil {
		p.CallLuaFunc(p.sidebarMenuFunc, lua.LString(cmd))
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
