package plugin

import (
	"fmt"

	"github.com/eugenioenko/ttt/internal/markdown"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

// reverseStyleMap maps term.Style constants to plugin style name strings.
var reverseStyleMap = map[term.Style]string{
	term.StyleDefault:         "default",
	term.StyleMuted:           "muted",
	term.StyleBorder:          "border",
	term.StyleSuccess:         "success",
	term.StyleDanger:          "danger",
	term.StyleWarning:         "warning",
	term.StyleSidebarSelected: "selected",
	term.StylePaletteItem:     "item",
	term.StyleLineNumber:      "line",
	term.StyleInput:           "input",
	term.StyleHoverBold:       "bold",
	term.StyleHoverCode:       "code",
	term.StyleSyntaxComment:   "syntax_comment",
	term.StyleSyntaxString:    "syntax_string",
	term.StyleSyntaxKeyword:   "syntax_keyword",
	term.StyleSyntaxNumber:    "syntax_number",
	term.StyleSyntaxOperator:  "syntax_operator",
	term.StyleSyntaxFunction:  "syntax_function",
	term.StyleSyntaxType:      "syntax_type",
	term.StyleSyntaxBuiltin:   "syntax_builtin",
	term.StyleSyntaxVariable:  "syntax_variable",
	term.StyleSyntaxTag:       "syntax_tag",
	term.StyleSyntaxAttribute: "syntax_attribute",
}

func styleToName(s term.Style) string {
	if name, ok := reverseStyleMap[s]; ok {
		return name
	}
	return "default"
}

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

		L.SetField(mod, "confirm", L.NewFunction(func(L *lua.LState) int {
			message := L.CheckString(1)
			callback := L.CheckFunction(2)
			if p.ShowConfirmDialog != nil {
				p.ShowConfirmDialog(message, func() {
					if err := p.CallLuaFunc(callback); err != nil {
						p.logError("confirm callback", err)
					}
				})
			}
			return 0
		}))

		L.SetField(mod, "open_drawer", L.NewFunction(func(L *lua.LState) int {
			if err := p.Granted.Check("panel.drawer"); err != nil {
				L.ArgError(1, "panel.drawer permission not granted")
				return 0
			}
			tbl := L.CheckTable(1)
			renderFunc, ok := L.GetField(tbl, "render").(*lua.LFunction)
			if !ok {
				L.ArgError(1, "render function required")
				return 0
			}
			width := 40
			minWidth := 20
			if w := L.GetField(tbl, "width"); w != lua.LNil {
				if n, ok := w.(lua.LNumber); ok {
					width = int(n)
				}
			}
			if mw := L.GetField(tbl, "min_width"); mw != lua.LNil {
				if n, ok := mw.(lua.LNumber); ok {
					minWidth = int(n)
				}
			}
			if p.OpenDrawer != nil {
				p.OpenDrawer(renderFunc, width, minWidth)
			}
			return 0
		}))

		L.SetField(mod, "close_drawer", L.NewFunction(func(L *lua.LState) int {
			if p.CloseDrawer != nil {
				p.CloseDrawer()
			}
			return 0
		}))

		L.SetField(mod, "open_tab", L.NewFunction(func(L *lua.LState) int {
			if err := p.Granted.Check("panel.editor"); err != nil {
				L.ArgError(1, "panel.editor permission not granted")
				return 0
			}
			tbl := L.CheckTable(1)
			title := "Plugin"
			if t := L.GetField(tbl, "title"); t != lua.LNil {
				title = t.String()
			}
			renderFunc, ok := L.GetField(tbl, "render").(*lua.LFunction)
			if !ok {
				L.ArgError(1, "render function required")
				return 0
			}
			var eventFunc *lua.LFunction
			if fn, ok := L.GetField(tbl, "on_event").(*lua.LFunction); ok {
				eventFunc = fn
			}
			if p.OpenTab != nil {
				p.OpenTab(title, renderFunc, eventFunc)
			}
			return 0
		}))

		L.SetField(mod, "close_tab", L.NewFunction(func(L *lua.LState) int {
			id := L.CheckString(1)
			if p.CloseTab != nil {
				p.CloseTab(id)
			}
			return 0
		}))

		L.SetField(mod, "click", L.NewFunction(func(L *lua.LState) int {
			x := L.CheckInt(1)
			y := L.CheckInt(2)
			if p.SimulateClick != nil {
				p.SimulateClick(x, y)
			}
			return 0
		}))

		L.SetField(mod, "drag", L.NewFunction(func(L *lua.LState) int {
			x1 := L.CheckInt(1)
			y1 := L.CheckInt(2)
			x2 := L.CheckInt(3)
			y2 := L.CheckInt(4)
			if p.SimulateDrag != nil {
				p.SimulateDrag(x1, y1, x2, y2)
			}
			return 0
		}))

		L.SetField(mod, "screenshot", L.NewFunction(func(L *lua.LState) int {
			path := L.CheckString(1)
			if p.ScreenshotToFile != nil {
				if err := p.ScreenshotToFile(path); err != nil {
					L.ArgError(1, err.Error())
				}
			}
			return 0
		}))

		L.SetField(mod, "debug", L.NewFunction(func(L *lua.LState) int {
			path := L.CheckString(1)
			if p.DebugDumpToFile != nil {
				if err := p.DebugDumpToFile(path); err != nil {
					L.ArgError(1, err.Error())
				}
			}
			return 0
		}))

		L.SetField(mod, "quit", L.NewFunction(func(L *lua.LState) int {
			if p.QuitApp != nil {
				p.QuitApp()
			}
			return 0
		}))

		L.SetField(mod, "markdown", L.NewFunction(func(L *lua.LState) int {
			text := L.CheckString(1)
			rendered := markdown.Render(text)
			result := L.NewTable()
			for i, line := range rendered {
				lineTable := L.NewTable()
				for j, span := range line.Spans {
					spanTable := L.NewTable()
					L.SetField(spanTable, "text", lua.LString(span.Text))
					L.SetField(spanTable, "style", lua.LString(styleToName(span.Style)))
					lineTable.RawSetInt(j+1, spanTable)
				}
				result.RawSetInt(i+1, lineTable)
			}
			L.Push(result)
			return 1
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
