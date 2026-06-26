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
	}
	return "unknown"
}

type WidgetDesc struct {
	Kind WidgetKind
	Key  string

	Text      string
	TextStyle string

	Items    []*widgets.TreeNode
	Indent   int
	OnSelect *lua.LFunction
	OnExpand *lua.LFunction

	Label   string
	OnClick *lua.LFunction

	Placeholder string
	Prefix      string
	OnChange    *lua.LFunction
	OnSubmit    *lua.LFunction
}
