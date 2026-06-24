package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TreeNode struct {
	ID         string      `json:"id"`
	Label      string      `json:"label"`
	Icon       string      `json:"icon,omitempty"`
	Badge      string      `json:"badge,omitempty"`
	BadgeStyle term.Style  `json:"-"`
	Children   []*TreeNode `json:"children,omitempty"`
	Actions    []Action    `json:"actions,omitempty"`
	Muted      bool        `json:"-"`
	Expandable bool        `json:"-"`

	Expanded bool `json:"-"`
	depth    int
}

type ListItem struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Icon    string   `json:"icon,omitempty"`
	Badge   string   `json:"badge,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

type Action struct {
	Icon    string `json:"icon"`
	Command string `json:"command"`
}

type MenuEntry struct {
	Label     string `json:"label"`
	Command   string `json:"command"`
	Separator bool   `json:"separator,omitempty"`
}

type TreeConfig struct {
	Items          []*TreeNode `json:"items"`
	NodeMenu       []MenuEntry `json:"nodeMenu,omitempty"`
	MenuIcon       string      `json:"menuIcon,omitempty"`
	MenuIconPadded bool        `json:"menuIconPadded,omitempty"`
	ActiveID       string      `json:"-"`

	OnCommand func(command string, node *TreeNode)
	OnMenu    func(entries []MenuEntry, node *TreeNode, screenX, screenY int)
	OnExpand  func(node *TreeNode)
}

type TreeWidget struct {
	BaseWidget
	Config   TreeConfig
	flatList []*TreeNode

	selected  int
	scrollTop int
	lastSel   int
	focused   bool

	scrollbar scrollbar
}

func NewTreeWidget(cfg TreeConfig) *TreeWidget {
	t := &TreeWidget{Config: cfg}
	t.flatten()
	return t
}

func (t *TreeWidget) Height() int { return 0 }
func (t *TreeWidget) Width() int  { return 0 }
func (t *TreeWidget) Focusable() bool    { return true }
func (t *TreeWidget) SetFocused(f bool)  { t.focused = f }
func (t *TreeWidget) IsFocused() bool    { return t.focused }

func (t *TreeWidget) Selected() *TreeNode {
	if t.selected >= 0 && t.selected < len(t.flatList) {
		return t.flatList[t.selected]
	}
	return nil
}

func (t *TreeWidget) SelectByID(id string) {
	for i, node := range t.flatList {
		if node.ID == id {
			t.selected = i
			return
		}
	}
}

func (t *TreeWidget) SetItems(items []*TreeNode) {
	t.Config.Items = items
	t.flatten()
	t.clampSelected()
}

func (t *TreeWidget) SetActiveID(id string) {
	t.Config.ActiveID = id
}

func (t *TreeWidget) Reload() {
	expanded := map[string]bool{}
	t.collectExpanded(t.Config.Items, expanded)
	for _, root := range t.Config.Items {
		if t.Config.OnExpand != nil {
			t.Config.OnExpand(root)
		}
		t.restoreExpanded(root, expanded)
	}
	t.flatten()
	t.clampSelected()
}

func (t *TreeWidget) collectExpanded(nodes []*TreeNode, out map[string]bool) {
	for _, node := range nodes {
		if node.Expanded {
			out[node.ID] = true
			t.collectExpanded(node.Children, out)
		}
	}
}

func (t *TreeWidget) restoreExpanded(node *TreeNode, expanded map[string]bool) {
	for _, child := range node.Children {
		if expanded[child.ID] && len(child.Children) > 0 {
			child.Expanded = true
			t.restoreExpanded(child, expanded)
		}
	}
}

func (t *TreeWidget) flatten() {
	t.flatList = nil
	for _, root := range t.Config.Items {
		t.flattenNode(root, 0)
	}
}

func (t *TreeWidget) flattenNode(node *TreeNode, depth int) {
	node.depth = depth
	t.flatList = append(t.flatList, node)
	if node.Expanded && len(node.Children) > 0 {
		for _, child := range node.Children {
			t.flattenNode(child, depth+1)
		}
	}
}

func (t *TreeWidget) clampSelected() {
	if t.selected >= len(t.flatList) {
		t.selected = len(t.flatList) - 1
	}
	if t.selected < 0 {
		t.selected = 0
	}
}

func (t *TreeWidget) ensureVisible(visibleH int) {
	if t.selected != t.lastSel {
		t.lastSel = t.selected
		if t.selected < t.scrollTop {
			t.scrollTop = t.selected
		}
		if t.selected >= t.scrollTop+visibleH {
			t.scrollTop = t.selected - visibleH + 1
		}
	}
}

func (t *TreeWidget) Render(surface Surface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	if h <= 0 || w <= 0 {
		return
	}

	t.ensureVisible(h)

	t.scrollbar.X = t.rect.X + w - 1
	t.scrollbar.Y = t.rect.Y
	t.scrollbar.Height = h
	t.scrollbar.TotalItems = len(t.flatList)
	t.scrollbar.TopItem = t.scrollTop

	for i := range h {
		idx := t.scrollTop + i
		if idx >= len(t.flatList) {
			break
		}
		node := t.flatList[idx]
		t.renderNode(surface, node, idx, i, w)
	}

	t.scrollbar.Render(surface, w-1, 0)
}

func (t *TreeWidget) renderNode(surface Surface, node *TreeNode, idx, y, w int) {
	style := term.StyleDefault
	if idx == t.selected {
		style = term.StyleSidebarSelected
	} else if t.Config.ActiveID != "" && node.ID == t.Config.ActiveID {
		style = term.StyleSidebarSelected
	}

	for x := range w {
		surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
	}

	x := node.depth * 2

	hasChildren := len(node.Children) > 0 || node.Expandable
	if hasChildren {
		chevron := '▶'
		if node.Expanded {
			chevron = '▼'
		}
		if x < w {
			surface.SetCell(x, y, term.Cell{Ch: chevron, Style: style})
		}
		x++
		if x < w {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}
		x++
	}

	if node.Icon != "" {
		for _, ch := range node.Icon {
			if x >= w-1 {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			x++
		}
		if x < w-1 {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
			x++
		}
	}

	labelStyle := style
	if node.Muted && idx != t.selected {
		labelStyle = term.StyleMuted
	}
	for _, ch := range node.Label {
		if x >= w-1 {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: labelStyle})
		x++
	}

	if node.Badge != "" {
		badgeStyle := node.BadgeStyle
		if badgeStyle == term.StyleDefault {
			badgeStyle = term.StyleMuted
		}
		if idx == t.selected {
			badgeStyle = style
		}
		x++
		for _, ch := range node.Badge {
			if x >= w-1 {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: badgeStyle})
			x++
		}
	}

	rightX := w - 2

	if len(t.Config.NodeMenu) > 0 {
		icon := t.Config.MenuIcon
		if icon == "" {
			if len(node.Children) > 0 {
				icon = "⋮"
			} else {
				icon = "⋯"
			}
		}
		dd := DropdownWidget{Config: DropdownConfig{Icon: icon, Padded: t.Config.MenuIconPadded}}
		dw := dd.Width()
		dd.Render(surface, rightX-dw+1, y, style)
		rightX -= dw
		if !t.Config.MenuIconPadded {
			if rightX > x && rightX < w {
				surface.SetCell(rightX, y, term.Cell{Ch: ' ', Style: style})
			}
			rightX--
		}
	}

	for i := len(node.Actions) - 1; i >= 0; i-- {
		action := node.Actions[i]
		for _, ch := range action.Icon {
			if rightX > x && rightX < w {
				surface.SetCell(rightX, y, term.Cell{Ch: ch, Style: style})
			}
			rightX--
		}
		if i > 0 {
			if rightX > x && rightX < w {
				surface.SetCell(rightX, y, term.Cell{Ch: ' ', Style: style})
			}
			rightX--
		}
	}
}

func (t *TreeWidget) HandleEvent(ev tcell.Event) bool {
	if newTop, consumed := t.scrollbar.HandleEvent(ev); consumed {
		t.scrollTop = newTop
		return true
	}

	switch tev := ev.(type) {
	case *tcell.EventMouse:
		return t.handleMouse(tev)
	case *tcell.EventKey:
		return t.handleKey(tev)
	}
	return false
}

func (t *TreeWidget) handleMouse(ev *tcell.EventMouse) bool {
	btn := ev.Buttons()
	mx, my := ev.Position()
	r := t.rect
	if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
		return false
	}

	if btn&tcell.WheelUp != 0 {
		t.scrollTop -= 3
		if t.scrollTop < 0 {
			t.scrollTop = 0
		}
		return true
	}
	if btn&tcell.WheelDown != 0 {
		max := len(t.flatList) - r.H
		if max < 0 {
			max = 0
		}
		t.scrollTop += 3
		if t.scrollTop > max {
			t.scrollTop = max
		}
		return true
	}

	idx := t.scrollTop + (my - r.Y)
	if idx < 0 || idx >= len(t.flatList) {
		return false
	}

	if btn&tcell.Button2 != 0 {
		t.selected = idx
		if t.Config.OnMenu != nil {
			t.Config.OnMenu(t.Config.NodeMenu, t.flatList[idx], mx, my)
		}
		return true
	}

	if btn&tcell.Button1 != 0 {
		node := t.flatList[idx]
		t.selected = idx

		if mx >= t.rect.X+t.rect.W-3 && len(t.Config.NodeMenu) > 0 {
			if t.Config.OnMenu != nil {
				t.Config.OnMenu(t.Config.NodeMenu, node, mx, my)
			}
			return true
		}

		for _, action := range node.Actions {
			actionX := t.rect.X + t.rect.W - 2 - len([]rune(action.Icon))
			if mx >= actionX && mx < actionX+len([]rune(action.Icon))+1 {
				if t.Config.OnCommand != nil {
					t.Config.OnCommand(action.Command, node)
				}
				return true
			}
		}

		t.activateSelected()
		return true
	}

	return false
}

func (t *TreeWidget) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if t.selected > 0 {
			t.selected--
		}
		return true
	case tcell.KeyDown:
		if t.selected < len(t.flatList)-1 {
			t.selected++
		}
		return true
	case tcell.KeyLeft:
		t.collapseSelected()
		return true
	case tcell.KeyRight:
		t.expandSelected()
		return true
	case tcell.KeyEnter:
		if ev.Modifiers()&tcell.ModShift != 0 {
			if t.Config.OnMenu != nil && t.selected >= 0 && t.selected < len(t.flatList) {
				t.Config.OnMenu(t.Config.NodeMenu, t.flatList[t.selected], t.rect.X, t.rect.Y+t.selected-t.scrollTop)
			}
			return true
		}
		t.activateSelected()
		return true
	case tcell.KeyRune:
		if ev.Rune() == ' ' {
			t.activateSelected()
			return true
		}
		return t.handleShortcutKey(ev.Rune())
	}
	return false
}

func (t *TreeWidget) handleShortcutKey(r rune) bool {
	node := t.Selected()
	if node == nil {
		return false
	}
	for _, action := range node.Actions {
		if len(action.Icon) == 1 && rune(action.Icon[0]) == r {
			if t.Config.OnCommand != nil {
				t.Config.OnCommand(action.Command, node)
			}
			return true
		}
	}
	return false
}

func (n *TreeNode) isExpandable() bool {
	return len(n.Children) > 0 || n.Expandable
}

func (t *TreeWidget) activateSelected() {
	if t.selected < 0 || t.selected >= len(t.flatList) {
		return
	}
	node := t.flatList[t.selected]
	if node.isExpandable() {
		node.Expanded = !node.Expanded
		if node.Expanded && t.Config.OnExpand != nil {
			t.Config.OnExpand(node)
		}
		t.flatten()
	} else if t.Config.OnCommand != nil {
		t.Config.OnCommand("activate", node)
	}
}

func (t *TreeWidget) collapseSelected() {
	if t.selected < 0 || t.selected >= len(t.flatList) {
		return
	}
	node := t.flatList[t.selected]
	if node.isExpandable() && node.Expanded {
		node.Expanded = false
		t.flatten()
	}
}

func (t *TreeWidget) expandSelected() {
	if t.selected < 0 || t.selected >= len(t.flatList) {
		return
	}
	node := t.flatList[t.selected]
	if node.isExpandable() && !node.Expanded {
		node.Expanded = true
		if t.Config.OnExpand != nil {
			t.Config.OnExpand(node)
		}
		t.flatten()
	}
}
