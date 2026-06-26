package plugin

import (
	"fmt"

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
	surface     widgets.Surface
	plugin      *Plugin
	descs       []WidgetDesc
	usedRaw     bool
	usedWidgets bool
	descCounts  map[WidgetKind]int
}

func NewPanelProxy(surface widgets.Surface, plugin *Plugin) *PanelProxy {
	return &PanelProxy{
		surface:    surface,
		plugin:     plugin,
		descCounts: make(map[WidgetKind]int),
	}
}

func (pp *PanelProxy) Descriptors() []WidgetDesc { return pp.descs }
func (pp *PanelProxy) UsedWidgets() bool         { return pp.usedWidgets }
func (pp *PanelProxy) UsedRaw() bool             { return pp.usedRaw }

func (pp *PanelProxy) appendDesc(kind WidgetKind, desc WidgetDesc) {
	idx := pp.descCounts[kind]
	pp.descCounts[kind] = idx + 1
	desc.Kind = kind
	desc.Key = fmt.Sprintf("%s:%d", kind, idx)
	pp.descs = append(pp.descs, desc)
	pp.usedWidgets = true
}

func RegisterPanelType(L *lua.LState) {
	mt := L.NewTypeMetatable(panelTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"size":   panelSize,
		"cell":   panelCell,
		"text":   panelText,
		"clear":  panelClear,
		"label":  panelLabelWidget,
		"tree":   panelTreeWidget,
		"list":   panelListWidget,
		"button": panelButtonWidget,
		"input":  panelInputWidget,
		"redraw": panelRedraw,
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

func resolveStyleName(name string) term.Style {
	if s, ok := styleMap[name]; ok {
		return s
	}
	return term.StyleDefault
}

// Raw cell API

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
	proxy.usedRaw = true
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
	proxy.usedRaw = true
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
	proxy.usedRaw = true
	x := int(L.CheckNumber(2))
	y := int(L.CheckNumber(3))
	w := int(L.CheckNumber(4))
	h := int(L.CheckNumber(5))

	proxy.surface.ClearRect(x, y, w, h, term.StyleDefault)
	return 0
}

// Widget helpers

func panelLabelWidget(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}

	desc := WidgetDesc{}
	arg := L.Get(2)

	switch v := arg.(type) {
	case lua.LString:
		desc.Text = string(v)
	case *lua.LTable:
		if t := L.GetField(v, "text"); t != lua.LNil {
			desc.Text = t.String()
		}
		if s := L.GetField(v, "style"); s != lua.LNil {
			desc.TextStyle = s.String()
		}
	default:
		desc.Text = arg.String()
	}

	proxy.appendDesc(WidgetLabel, desc)
	return 0
}

func panelTreeWidget(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}

	tbl := L.CheckTable(2)
	desc := WidgetDesc{Indent: 2}

	if items, ok := L.GetField(tbl, "items").(*lua.LTable); ok {
		desc.Items = LuaTableToTreeNodes(L, items)
	}
	if v := L.GetField(tbl, "indent"); v != lua.LNil {
		desc.Indent = int(lua.LVAsNumber(v))
	}
	if fn, ok := L.GetField(tbl, "on_select").(*lua.LFunction); ok {
		desc.OnSelect = fn
	}
	if fn, ok := L.GetField(tbl, "on_expand").(*lua.LFunction); ok {
		desc.OnExpand = fn
	}

	proxy.appendDesc(WidgetTree, desc)
	return 0
}

func panelListWidget(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}

	tbl := L.CheckTable(2)
	desc := WidgetDesc{}

	if items, ok := L.GetField(tbl, "items").(*lua.LTable); ok {
		desc.Items = LuaTableToTreeNodes(L, items)
	}
	if fn, ok := L.GetField(tbl, "on_select").(*lua.LFunction); ok {
		desc.OnSelect = fn
	}

	proxy.appendDesc(WidgetList, desc)
	return 0
}

func panelButtonWidget(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}

	tbl := L.CheckTable(2)
	desc := WidgetDesc{}

	if v := L.GetField(tbl, "label"); v != lua.LNil {
		desc.Label = v.String()
	}
	if fn, ok := L.GetField(tbl, "on_click").(*lua.LFunction); ok {
		desc.OnClick = fn
	}

	proxy.appendDesc(WidgetButton, desc)
	return 0
}

func panelInputWidget(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}

	tbl := L.CheckTable(2)
	desc := WidgetDesc{}

	if v := L.GetField(tbl, "placeholder"); v != lua.LNil {
		desc.Placeholder = v.String()
	}
	if v := L.GetField(tbl, "prefix"); v != lua.LNil {
		desc.Prefix = v.String()
	}
	if fn, ok := L.GetField(tbl, "on_change").(*lua.LFunction); ok {
		desc.OnChange = fn
	}
	if fn, ok := L.GetField(tbl, "on_submit").(*lua.LFunction); ok {
		desc.OnSubmit = fn
	}

	proxy.appendDesc(WidgetInput, desc)
	return 0
}

func panelRedraw(L *lua.LState) int {
	proxy := checkPanelProxy(L)
	if proxy == nil {
		return 0
	}
	if proxy.plugin != nil && proxy.plugin.RequestRedraw != nil {
		proxy.plugin.RequestRedraw()
	}
	return 0
}
