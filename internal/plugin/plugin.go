package plugin

import (
	"log/slog"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

type Plugin struct {
	Name     string
	Dir      string
	Manifest Manifest
	Granted  PermissionSet
	Enabled  bool
	State    *lua.LState

	SidebarTitle string
	RenderFunc   *lua.LFunction
	EventFunc    *lua.LFunction

	LastError error
}

func (p *Plugin) Init() error {
	p.State = NewSandbox()
	setupTTTModule(p.State, p)

	entry := filepath.Join(p.Dir, p.Manifest.Entry)
	if err := p.State.DoFile(entry); err != nil {
		p.LastError = err
		p.State.Close()
		p.State = nil
		slog.Error("plugin init failed", "plugin", p.Name, "error", err)
		return err
	}

	p.Enabled = true
	return nil
}

func (p *Plugin) Destroy() {
	if p.State != nil {
		p.State.Close()
		p.State = nil
	}
	p.Enabled = false
	p.RenderFunc = nil
	p.EventFunc = nil
}

func (p *Plugin) CallRender(proxy *PanelProxy) error {
	if p.State == nil || p.RenderFunc == nil {
		return nil
	}

	ud := PushPanelProxy(p.State, proxy)
	err := p.State.CallByParam(lua.P{
		Fn:      p.RenderFunc,
		NRet:    0,
		Protect: true,
	}, ud)
	if err != nil {
		p.LastError = err
		slog.Error("plugin render error", "plugin", p.Name, "error", err)
	}
	return err
}
