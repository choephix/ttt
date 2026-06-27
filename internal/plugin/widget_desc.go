package plugin

import (
	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

type WidgetKind int

const (
	WidgetLabel WidgetKind = iota
	WidgetTree
	WidgetList
	WidgetButton
	WidgetInput
	WidgetVStack
	WidgetBox
	WidgetDropdown
	WidgetTitle
	WidgetKeyValue
)

func (k WidgetKind) String() string {
	switch k {
	case WidgetLabel:
		return "label"
	case WidgetTree:
		return "tree"
	case WidgetList:
		return "list"
	case WidgetButton:
		return "button"
	case WidgetInput:
		return "input"
	case WidgetVStack:
		return "vstack"
	case WidgetBox:
		return "box"
	case WidgetDropdown:
		return "dropdown"
	case WidgetTitle:
		return "title"
	case WidgetKeyValue:
		return "keyvalue"
	}
	return "unknown"
}

type WidgetDesc struct {
	Kind WidgetKind
	Key  string

	Text      string
	TextStyle string
	Badge     string

	MarginTop    int
	MarginBottom int
	MarginLeft   int
	MarginRight  int
	PaddingTop    int
	PaddingBottom int
	PaddingLeft   int
	PaddingRight  int

	Items     []*widgets.TreeNode
	Indent    int
	OnSelect  *lua.LFunction
	OnExpand  *lua.LFunction
	OnCommand *lua.LFunction
	NodeMenu  []widgets.MenuEntry

	Label   string
	OnClick *lua.LFunction

	Placeholder string
	Prefix      string
	OnChange    *lua.LFunction
	OnSubmit    *lua.LFunction

	Children    []WidgetDesc
	Border      bool
	FixedHeight int
	Gap         int

	Entries         []widgets.MenuEntry
	OnMenu         *lua.LFunction
	KeyValueEntries []widgets.KeyValueEntry
}
