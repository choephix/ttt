package plugin

import (
	"fmt"

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
