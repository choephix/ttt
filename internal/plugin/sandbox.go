package plugin

import (
	"fmt"

	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

func NewSandbox() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})

	for _, pair := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage},
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(pair.fn))
		L.Push(lua.LString(pair.name))
		L.Call(1, 0)
	}

	for _, name := range []string{"dofile", "loadfile"} {
		L.SetGlobal(name, lua.LNil)
	}

	return L
}

func setupTTTModule(L *lua.LState, p *Plugin) {
	RegisterPanelType(L)

	loader := func(L *lua.LState) int {
		mod := L.NewTable()

		L.SetField(mod, "register", L.NewFunction(func(L *lua.LState) int {
			tbl := L.CheckTable(1)

			sidebar := L.GetField(tbl, "sidebar")
			if st, ok := sidebar.(*lua.LTable); ok {
				if err := p.Granted.Check("panel.sidebar"); err != nil {
					L.ArgError(1, "panel.sidebar permission not granted")
					return 0
				}
				if title := L.GetField(st, "title"); title != lua.LNil {
					p.SidebarTitle = title.String()
				}
				if fn, ok := L.GetField(st, "render").(*lua.LFunction); ok {
					p.RenderFunc = fn
				}
				if fn, ok := L.GetField(st, "on_event").(*lua.LFunction); ok {
					p.EventFunc = fn
				}
				if actions, ok := L.GetField(st, "actions").(*lua.LTable); ok {
					p.SidebarMenuEntries = parseLuaMenuEntries(L, actions)
				}
				if fn, ok := L.GetField(st, "on_action").(*lua.LFunction); ok {
					p.SidebarMenuFunc = fn
				}
			}

			bottom := L.GetField(tbl, "bottom")
			if bt, ok := bottom.(*lua.LTable); ok {
				if err := p.Granted.Check("panel.bottom"); err != nil {
					L.ArgError(1, "panel.bottom permission not granted")
					return 0
				}
				if title := L.GetField(bt, "title"); title != lua.LNil {
					p.BottomTitle = title.String()
				}
				if fn, ok := L.GetField(bt, "render").(*lua.LFunction); ok {
					p.BottomRenderFunc = fn
				}
				if fn, ok := L.GetField(bt, "on_event").(*lua.LFunction); ok {
					p.BottomEventFunc = fn
				}
			}

			commands := L.GetField(tbl, "commands")
			if ct, ok := commands.(*lua.LTable); ok {
				if err := p.Granted.Check("commands"); err != nil {
					L.ArgError(1, "commands permission not granted")
					return 0
				}
				ct.ForEach(func(_ lua.LValue, v lua.LValue) {
					entry, ok := v.(*lua.LTable)
					if !ok {
						return
					}
					id := L.GetField(entry, "id")
					title := L.GetField(entry, "title")
					handler, hOk := L.GetField(entry, "handler").(*lua.LFunction)
					if id == lua.LNil || title == lua.LNil || !hOk {
						return
					}
					p.Commands = append(p.Commands, PluginCommand{
						ID:      id.String(),
						Title:   title.String(),
						Handler: handler,
					})
				})
			}

			keybindings := L.GetField(tbl, "keybindings")
			if kt, ok := keybindings.(*lua.LTable); ok {
				if err := p.Granted.Check("keybindings"); err != nil {
					L.ArgError(1, "keybindings permission not granted")
					return 0
				}
				kt.ForEach(func(_ lua.LValue, v lua.LValue) {
					entry, ok := v.(*lua.LTable)
					if !ok {
						return
					}
					key := L.GetField(entry, "key")
					cmd := L.GetField(entry, "command")
					if key == lua.LNil || cmd == lua.LNil {
						return
					}
					p.PluginKeybindings = append(p.PluginKeybindings, PluginKeybinding{
						Key:     key.String(),
						Command: cmd.String(),
					})
				})
			}

			return 0
		}))

		L.SetField(mod, "log", L.NewFunction(func(L *lua.LState) int {
			nargs := L.GetTop()
			var level, message string
			if nargs >= 2 {
				level = L.CheckString(1)
				message = L.CheckString(2)
			} else {
				level = "info"
				message = L.CheckString(1)
			}
			if p.Log != nil {
				p.Log(level, message)
			}
			return 0
		}))

		L.SetField(mod, "show_info", L.NewFunction(func(L *lua.LState) int {
			title := L.CheckString(1)
			tbl := L.CheckTable(2)
			var entries []widgets.KeyValueEntry
			tbl.ForEach(func(_, v lua.LValue) {
				row, ok := v.(*lua.LTable)
				if !ok {
					return
				}
				entry := widgets.KeyValueEntry{}
				if k := L.GetField(row, "key"); k != lua.LNil {
					entry.Key = k.String()
				}
				if val := L.GetField(row, "value"); val != lua.LNil {
					entry.Value = val.String()
				}
				entries = append(entries, entry)
			})
			if p.ShowInfoDialog != nil {
				p.ShowInfoDialog(title, entries)
			}
			return 0
		}))

		L.Push(mod)
		return 1
	}

	L.PreloadModule("ttt", loader)

	allowedModules := map[string]bool{
		"ttt":        true,
		"ttt.editor": true,
		"ttt.fs":     true,
		"ttt.system": true,
		"ttt.net":    true,
		"ttt.events": true,
	}

	origRequire := L.GetGlobal("require")
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		if !allowedModules[name] {
			L.ArgError(1, fmt.Sprintf("module %q is not available", name))
			return 0
		}
		L.Push(origRequire)
		L.Push(lua.LString(name))
		L.Call(1, 1)
		return 1
	}))
}
