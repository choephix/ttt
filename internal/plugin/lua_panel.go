package plugin

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

const panelTypeName = "panel"

var styleMap = map[string]term.Style{
	"default":  term.StyleDefault,
	"muted":    term.StyleMuted,
	"border":   term.StyleBorder,
	"success":  term.StyleSuccess,
	"danger":   term.StyleDanger,
	"warning":  term.StyleWarning,
	"selected": term.StyleSidebarSelected,
	"item":     term.StylePaletteItem,
	"line":     term.StyleLineNumber,
	"input":    term.StyleInput,
}

type PanelProxy struct {
	surface widgets.Surface
}

func NewPanelProxy(surface widgets.Surface) *PanelProxy {
	return &PanelProxy{surface: surface}
}

func RegisterPanelType(L *lua.LState) {
	mt := L.NewTypeMetatable(panelTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"size":  panelSize,
		"cell":  panelCell,
		"text":  panelText,
		"clear": panelClear,
	}))
}

func PushPanelProxy(L *lua.LState, proxy *PanelProxy) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = proxy
	L.SetMetatable(ud, L.GetTypeMetatable(panelTypeName))
	return ud
}

func checkPanelProxy(L *lua.LState) *PanelProxy {
	ud := L.CheckUserData(1)
	if proxy, ok := ud.Value.(*PanelProxy); ok {
		return proxy
	}
	L.ArgError(1, "panel expected")
	return nil
}

func resolveStyle(L *lua.LState, argPos int) term.Style {
	v := L.Get(argPos)
	if v == nil || v == lua.LNil {
		return term.StyleDefault
	}

	if tbl, ok := v.(*lua.LTable); ok {
		if s := L.GetField(tbl, "style"); s != lua.LNil {
			if mapped, ok := styleMap[s.String()]; ok {
				return mapped
			}
		}
	}

	if str, ok := v.(lua.LString); ok {
		if mapped, ok := styleMap[string(str)]; ok {
			return mapped
		}
	}

	return term.StyleDefault
}

func panelSize(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}
	w, h := proxy.surface.Size()
	L.Push(lua.LNumber(w))
	L.Push(lua.LNumber(h))
	return 2
}

func panelCell(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}
	x := int(L.CheckNumber(2))
	y := int(L.CheckNumber(3))
	ch := L.CheckString(4)
	style := resolveStyle(L, 5)

	runes := []rune(ch)
	if len(runes) > 0 {
		proxy.surface.SetCell(x, y, term.Cell{Ch: runes[0], Style: style})
	}
	return 0
}

func panelText(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}
	x := int(L.CheckNumber(2))
	y := int(L.CheckNumber(3))
	text := L.CheckString(4)
	style := resolveStyle(L, 5)

	w, _ := proxy.surface.Size()
	proxy.surface.DrawText(x, y, text, w-x, style)
	return 0
}

func panelClear(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}
	x := int(L.CheckNumber(2))
	y := int(L.CheckNumber(3))
	w := int(L.CheckNumber(4))
	h := int(L.CheckNumber(5))

	proxy.surface.ClearRect(x, y, w, h, term.StyleDefault)
	return 0
}
