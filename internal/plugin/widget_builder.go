package plugin

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	lua "github.com/yuin/gopher-lua"
)

type WidgetState struct {
	keys    []string
	items   []widgets.Widget
	root    *widgets.VStackWidget
	focus   *widgets.FocusManager
}

func NewWidgetState() *WidgetState {
	return &WidgetState{
		focus: widgets.NewFocusManager(),
	}
}

func (ws *WidgetState) Reconcile(descs []WidgetDesc, p *Plugin) *widgets.VStackWidget {
	newKeys := make([]string, len(descs))
	newWidgets := make([]widgets.Widget, len(descs))

	for i, desc := range descs {
		newKeys[i] = desc.Key

		if i < len(ws.keys) && ws.keys[i] == desc.Key {
			updateWidget(ws.items[i], desc, p)
			newWidgets[i] = ws.items[i]
		} else {
			newWidgets[i] = createWidget(desc, p)
		}
	}

	ws.keys = newKeys
	ws.items = newWidgets
	ws.root = widgets.NewVStackWidget(newWidgets...)
	ws.focus.Collect(ws.root)
	return ws.root
}

func createWidget(desc WidgetDesc, p *Plugin) widgets.Widget {
	switch desc.Kind {
	case WidgetLabel:
		return createLabelWidget(desc)
	case WidgetTree:
		return createTreeWidget(desc, p)
	case WidgetList:
		return createListWidget(desc, p)
	case WidgetButton:
		return createButtonWidget(desc, p)
	case WidgetInput:
		return createInputWidget(desc, p)
	case WidgetVStack:
		return createVStackWidget(desc, p)
	case WidgetBox:
		return createBoxWidget(desc, p)
	case WidgetDropdown:
		return createDropdownWidget(desc, p)
	}
	return widgets.NewLabelWidget(widgets.LabelConfig{Text: "unknown widget"})
}

func updateWidget(w widgets.Widget, desc WidgetDesc, p *Plugin) {
	switch desc.Kind {
	case WidgetLabel:
		if lw, ok := w.(*widgets.LabelWidget); ok {
			lw.Config.Text = desc.Text
			if desc.TextStyle != "" {
				lw.Config.Style = resolveStyleName(desc.TextStyle)
			}
		}
	case WidgetTree, WidgetList:
		if tw, ok := w.(*widgets.TreeWidget); ok {
			expanded := map[string]bool{}
			tw.CollectExpanded(expanded)
			for _, item := range tw.Config.Items {
				if item.Expanded {
					expanded[item.ID] = true
				}
			}
			tw.SetItems(desc.Items)
			tw.RestoreExpanded(expanded)
			for _, item := range tw.Config.Items {
				if expanded[item.ID] && item.Expandable {
					item.Expanded = true
				}
			}
			wireTreeCallbacks(tw, desc, p)
		}
	case WidgetButton:
		if bw, ok := w.(*widgets.ButtonWidget); ok {
			_ = bw
			// Button label cannot be changed after construction due to accelerator parsing.
			// Rewire callback only.
			wireButtonCallback(bw, desc, p)
		}
	case WidgetInput:
		if iw, ok := w.(*widgets.InputWidget); ok {
			iw.Config.Placeholder = desc.Placeholder
			wireInputCallbacks(iw, desc, p)
		}
	case WidgetVStack:
		if vs, ok := w.(*widgets.VStackWidget); ok {
			children := make([]widgets.Widget, len(desc.Children))
			for i, cd := range desc.Children {
				if i < len(vs.Children) {
					updateWidget(vs.Children[i], cd, p)
					children[i] = vs.Children[i]
				} else {
					children[i] = createWidget(cd, p)
				}
			}
			vs.Children = children
		}
	case WidgetBox:
		if bw, ok := w.(*widgets.BoxWidget); ok {
			if len(desc.Children) > 0 {
				if vs, ok := bw.Child.(*widgets.VStackWidget); ok {
					children := make([]widgets.Widget, len(desc.Children))
					for i, cd := range desc.Children {
						if i < len(vs.Children) {
							updateWidget(vs.Children[i], cd, p)
							children[i] = vs.Children[i]
						} else {
							children[i] = createWidget(cd, p)
						}
					}
					vs.Children = children
				} else {
					bw.Child = createVStackFromDescs(desc.Children, p)
				}
			}
		}
	}
}

func createLabelWidget(desc WidgetDesc) *widgets.LabelWidget {
	style := term.StyleDefault
	if desc.TextStyle != "" {
		style = resolveStyleName(desc.TextStyle)
	}
	return widgets.NewLabelWidget(widgets.LabelConfig{
		Text:  desc.Text,
		Style: style,
	})
}

func createTreeWidget(desc WidgetDesc, p *Plugin) *widgets.TreeWidget {
	tw := widgets.NewTreeWidget(widgets.TreeConfig{
		Items:    desc.Items,
		Indent:   desc.Indent,
		NodeMenu: desc.NodeMenu,
	})
	wireTreeCallbacks(tw, desc, p)
	return tw
}

func createListWidget(desc WidgetDesc, p *Plugin) *widgets.TreeWidget {
	tw := widgets.NewTreeWidget(widgets.TreeConfig{
		Items:    desc.Items,
		NodeMenu: desc.NodeMenu,
	})
	wireTreeCallbacks(tw, desc, p)
	return tw
}

func createVStackFromDescs(descs []WidgetDesc, p *Plugin) *widgets.VStackWidget {
	children := make([]widgets.Widget, len(descs))
	for i, cd := range descs {
		children[i] = createWidget(cd, p)
	}
	return widgets.NewVStackWidget(children...)
}

func createVStackWidget(desc WidgetDesc, p *Plugin) *widgets.VStackWidget {
	vs := createVStackFromDescs(desc.Children, p)
	vs.Gap = desc.Gap
	return vs
}

func createBoxWidget(desc WidgetDesc, p *Plugin) *widgets.BoxWidget {
	var box *widgets.BoxWidget
	if desc.Border {
		borders := term.SingleBorderSet()
		if p.Borders != nil {
			borders = *p.Borders
		}
		box = widgets.NewBoxWithBorder(borders)
	} else {
		box = widgets.NewBoxWidget(widgets.BoxModel{})
	}
	if desc.FixedHeight > 0 {
		box.FixedHeight = desc.FixedHeight
	}
	if len(desc.Children) > 0 {
		box.Child = createVStackFromDescs(desc.Children, p)
	}
	return box
}

func createDropdownWidget(desc WidgetDesc, p *Plugin) *widgets.DropdownWidget {
	dd := widgets.NewDropdownWidget(widgets.DropdownConfig{
		Label:   desc.Label,
		Entries: desc.Entries,
		Box:     &widgets.BoxModel{PaddingLeft: 1, PaddingRight: 1},
	})
	wireDropdownCallback(dd, desc, p)
	return dd
}

func wireDropdownCallback(dd *widgets.DropdownWidget, desc WidgetDesc, p *Plugin) {
	if p.ShowContextMenu != nil && len(desc.Entries) > 0 {
		dd.Config.OnMenu = func(entries []widgets.MenuEntry, screenX, screenY int) {
			p.ShowContextMenu(entries, screenX, screenY, func(cmd string) {
				if desc.OnMenu != nil {
					if p.State != nil {
						p.CallLuaFunc(desc.OnMenu, lua.LString(cmd))
					}
				}
			})
		}
	}
}

func wireTreeCallbacks(tw *widgets.TreeWidget, desc WidgetDesc, p *Plugin) {
	if desc.OnSelect != nil {
		fn := desc.OnSelect
		tw.Config.OnSelect = func(node *widgets.TreeNode) {
			if p.State != nil {
				tbl := TreeNodeToLua(p.State, node)
				p.CallLuaFunc(fn, tbl)
			}
		}
	}
	if desc.OnExpand != nil {
		fn := desc.OnExpand
		tw.Config.OnExpand = func(node *widgets.TreeNode) {
			if p.State != nil {
				tbl := TreeNodeToLua(p.State, node)
				p.CallLuaFunc(fn, tbl)
			}
		}
	}
	if desc.OnCommand != nil {
		fn := desc.OnCommand
		tw.Config.OnCommand = func(command string, node *widgets.TreeNode) {
			if p.State != nil {
				tbl := TreeNodeToLua(p.State, node)
				p.CallLuaFunc(fn, lua.LString(command), tbl)
			}
		}
	}
	if len(desc.NodeMenu) > 0 {
		tw.Config.NodeMenu = desc.NodeMenu
		if p.ShowContextMenu != nil {
			tw.Config.OnMenu = func(entries []widgets.MenuEntry, node *widgets.TreeNode, sx, sy int) {
				p.ShowContextMenu(entries, sx, sy, func(cmd string) {
					if tw.Config.OnCommand != nil {
						tw.Config.OnCommand(cmd, node)
					}
				})
			}
		}
	}
}

func createButtonWidget(desc WidgetDesc, p *Plugin) *widgets.ButtonWidget {
	bw := widgets.NewButtonWidget(widgets.ButtonConfig{
		Label: desc.Label,
	})
	wireButtonCallback(bw, desc, p)
	return bw
}

func wireButtonCallback(bw *widgets.ButtonWidget, desc WidgetDesc, p *Plugin) {
	if desc.OnClick != nil {
		fn := desc.OnClick
		bw.Config.OnClick = func() {
			p.CallLuaFunc(fn)
		}
	}
}

func createInputWidget(desc WidgetDesc, p *Plugin) *widgets.InputWidget {
	iw := widgets.NewInputWidget(widgets.InputConfig{
		Placeholder: desc.Placeholder,
		Prefix:      desc.Prefix,
	})
	wireInputCallbacks(iw, desc, p)
	return iw
}

func wireInputCallbacks(iw *widgets.InputWidget, desc WidgetDesc, p *Plugin) {
	if desc.OnChange != nil {
		fn := desc.OnChange
		iw.Config.OnChange = func(text string) {
			if p.State != nil {
				p.CallLuaFunc(fn, lua.LString(text))
			}
		}
	}
	if desc.OnSubmit != nil {
		fn := desc.OnSubmit
		iw.Config.OnSubmit = func(text string) {
			if p.State != nil {
				p.CallLuaFunc(fn, lua.LString(text))
			}
		}
	}
}
