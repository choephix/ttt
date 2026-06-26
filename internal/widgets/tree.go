package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TreeNode struct {
	ID         string      `json:"id"`
	Label      string      `json:"label"`
	Icon       string      `json:"icon,omitempty"`
	IconStyle  term.Style  `json:"-"`
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
	Indent         int         `json:"indent,omitempty"`
	ActiveID       string      `json:"-"`
	EmptyText      string      `json:"emptyText,omitempty"`

	OnCommand func(command string, node *TreeNode)
	OnMenu    func(entries []MenuEntry, node *TreeNode, screenX, screenY int)
	OnExpand  func(node *TreeNode)
	OnSelect  func(node *TreeNode)
	OnKey     func(ev *tcell.EventKey, node *TreeNode) bool
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

func NewListWidget(items []ListItem) *TreeWidget {
	nodes := make([]*TreeNode, len(items))
	for i, li := range items {
		nodes[i] = &TreeNode{
			ID:      li.ID,
			Label:   li.Label,
			Icon:    li.Icon,
			Badge:   li.Badge,
			Actions: li.Actions,
		}
	}
	return NewTreeWidget(TreeConfig{Items: nodes})
}

func NewTreeWidget(cfg TreeConfig) *TreeWidget {
	if cfg.Indent == 0 {
		cfg.Indent = 2
	} else if cfg.Indent < 0 {
		cfg.Indent = 0
	}
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

	if len(t.flatList) == 0 && t.Config.EmptyText != "" {
		x := 1
		for _, ch := range t.Config.EmptyText {
			if x >= w {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: term.StyleDefault})
			x++
		}
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

func (t *TreeWidget) rightSideWidth(node *TreeNode) int {
	rw := 0
	if len(t.Config.NodeMenu) > 0 {
		label := t.Config.MenuIcon
		if label == "" {
			label = "⋮"
		}
		if t.Config.MenuIconPadded {
			rw += len([]rune(label)) + 2
		} else {
			rw += len([]rune(label)) + 2
		}
	}
	for i, action := range node.Actions {
		rw += len([]rune(action.Icon))
		if i > 0 {
			rw++
		}
	}
	return rw
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

	maxX := w - 2 - t.rightSideWidth(node)

	x := node.depth * t.Config.Indent

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
		iconStyle := node.IconStyle
		if iconStyle == term.StyleDefault {
			iconStyle = style
		}
		if idx == t.selected {
			iconStyle = style
		}
		for _, ch := range node.Icon {
			if x >= maxX {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: iconStyle})
			x++
		}
		if x < maxX {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
			x++
		}
	}

	labelStyle := style
	if node.Muted && idx != t.selected {
		labelStyle = term.StyleMuted
	}
	for _, ch := range node.Label {
		if x >= maxX {
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
			if x >= maxX {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: badgeStyle})
			x++
		}
	}

	rightX := w - 2

	if len(t.Config.NodeMenu) > 0 {
		label := t.Config.MenuIcon
		if label == "" {
			if len(node.Children) > 0 {
				label = "⋮"
			} else {
				label = "⋯"
			}
		}
		var box *BoxModel
		if t.Config.MenuIconPadded {
			box = &BoxModel{PaddingLeft: 1, PaddingRight: 1}
		} else {
			box = &BoxModel{PaddingLeft: 0, PaddingRight: 0}
		}
		dd := NewDropdownWidget(DropdownConfig{Label: label, Style: style, Box: box})
		dw := dd.Width()
		ddX := rightX - dw + 1
		dd.SetRect(Rect{X: ddX, Y: y, W: dw, H: 1})
		ddSurface := surface.Sub(Rect{X: ddX, Y: y, W: dw, H: 1})
		dd.Render(ddSurface)
		rightX -= dw
		if !t.Config.MenuIconPadded {
			if rightX >= 0 && rightX < w {
				surface.SetCell(rightX, y, term.Cell{Ch: ' ', Style: style})
			}
			rightX--
		}
	}

	for i := len(node.Actions) - 1; i >= 0; i-- {
		action := node.Actions[i]
		for _, ch := range action.Icon {
			if rightX >= 0 && rightX < w {
				surface.SetCell(rightX, y, term.Cell{Ch: ch, Style: style})
			}
			rightX--
		}
		if i > 0 {
			if rightX >= 0 && rightX < w {
				surface.SetCell(rightX, y, term.Cell{Ch: ' ', Style: style})
			}
			rightX--
		}
	}
}

func (t *TreeWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := t.scrollbar.HandleEvent(ev); consumed {
		t.scrollTop = newTop
		return EventConsumed
	}

	prev := t.selected
	var result EventResult
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		result = t.handleMouse(tev)
	case *tcell.EventKey:
		result = t.handleKey(tev)
	default:
		return EventIgnored
	}
	if t.selected != prev && t.Config.OnSelect != nil {
		t.Config.OnSelect(t.Selected())
	}
	return result
}

func (t *TreeWidget) handleMouse(ev *tcell.EventMouse) EventResult {
	btn := ev.Buttons()
	mx, my := ev.Position()
	r := t.rect
	if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
		return EventIgnored
	}

	if btn&tcell.WheelUp != 0 {
		t.scrollTop -= 3
		if t.scrollTop < 0 {
			t.scrollTop = 0
		}
		return EventConsumed
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
		return EventConsumed
	}

	idx := t.scrollTop + (my - r.Y)
	if idx < 0 || idx >= len(t.flatList) {
		return EventIgnored
	}

	if btn&tcell.Button2 != 0 {
		t.selected = idx
		if t.Config.OnMenu != nil {
			t.Config.OnMenu(t.Config.NodeMenu, t.flatList[idx], mx, my)
		}
		return EventConsumed
	}

	if btn&tcell.Button1 != 0 {
		node := t.flatList[idx]
		t.selected = idx

		if mx >= t.rect.X+t.rect.W-3 && len(t.Config.NodeMenu) > 0 {
			if t.Config.OnMenu != nil {
				t.Config.OnMenu(t.Config.NodeMenu, node, mx, my)
			}
			return EventConsumed
		}

		rightX := t.rect.X + t.rect.W - 2
		for i := len(node.Actions) - 1; i >= 0; i-- {
			action := node.Actions[i]
			iconW := len([]rune(action.Icon))
			actionX := rightX - iconW + 1
			if mx >= actionX && mx <= rightX {
				if t.Config.OnCommand != nil {
					t.Config.OnCommand(action.Command, node)
				}
				return EventConsumed
			}
			rightX = actionX - 1
			if i > 0 {
				rightX--
			}
		}

		t.ActivateSelected()
		return EventConsumed
	}

	return EventIgnored
}

func (t *TreeWidget) handleKey(ev *tcell.EventKey) EventResult {
	switch ev.Key() {
	case tcell.KeyUp:
		if t.selected > 0 {
			t.selected--
		}
		return EventConsumed
	case tcell.KeyDown:
		if t.selected < len(t.flatList)-1 {
			t.selected++
		}
		return EventConsumed
	case tcell.KeyLeft:
		t.collapseSelected()
		return EventConsumed
	case tcell.KeyRight:
		t.expandSelected()
		return EventConsumed
	case tcell.KeyEnter:
		if ev.Modifiers()&tcell.ModShift != 0 {
			if t.Config.OnMenu != nil && t.selected >= 0 && t.selected < len(t.flatList) {
				t.Config.OnMenu(t.Config.NodeMenu, t.flatList[t.selected], t.rect.X, t.rect.Y+t.selected-t.scrollTop)
			}
			return EventConsumed
		}
		t.ActivateSelected()
		return EventConsumed
	case tcell.KeyRune:
		if t.Config.OnKey != nil && t.Config.OnKey(ev, t.Selected()) {
			return EventConsumed
		}
		if ev.Rune() == ' ' {
			t.ActivateSelected()
			return EventConsumed
		}
		if t.handleShortcutKey(ev.Rune()) == EventConsumed {
			return EventConsumed
		}
	}
	return EventIgnored
}

func (t *TreeWidget) handleShortcutKey(r rune) EventResult {
	node := t.Selected()
	if node == nil {
		return EventIgnored
	}
	for _, action := range node.Actions {
		iconRunes := []rune(action.Icon)
		if len(iconRunes) == 1 && iconRunes[0] == r {
			if t.Config.OnCommand != nil {
				t.Config.OnCommand(action.Command, node)
			}
			return EventConsumed
		}
	}
	return EventIgnored
}

func (n *TreeNode) isExpandable() bool {
	return len(n.Children) > 0 || n.Expandable
}

func (t *TreeWidget) FlatList() []*TreeNode       { return t.flatList }
func (t *TreeWidget) SelectedIndex() int           { return t.selected }
func (t *TreeWidget) SetSelectedIndex(i int)       { t.selected = i; t.clampSelected() }
func (t *TreeWidget) ScrollTop() int               { return t.scrollTop }
func (t *TreeWidget) ItemCount() int               { return len(t.flatList) }

func (t *TreeWidget) ActivateSelected() {
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
