package ui

import (
	"fmt"
	"path/filepath"
	"ttt/internal/git"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ChangesGroup struct {
	Dir             string
	Name            string
	Staged          []git.FileStatus
	Unstaged        []git.FileStatus
	Expanded        bool
	StagedExpanded  bool
	ChangesExpanded bool
	Input           *InputWidget
}

type changesItemKind int

const (
	itemFile changesItemKind = iota
	itemHeader
	itemInput
	itemBorder
	itemSection
)

type changesItem struct {
	kind       changesItemKind
	groupIndex int
	fileIndex  int
	staged     bool
}

type ChangesWidget struct {
	BaseWidget
	SelectableList
	Dirs         []string
	Groups       []ChangesGroup
	items        []changesItem
	multiRoot    bool
	inputFocused bool
	OnOpenDiff   func(dir string, status git.FileStatus)
	OnRightClick func(dir string, status git.FileStatus, screenX, screenY int)
	OnCommit     func(dir string, message string)
}

func NewChangesWidget(dirs ...string) *ChangesWidget {
	w := &ChangesWidget{
		Dirs:      dirs,
		multiRoot: len(dirs) > 1,
	}
	w.Refresh()
	return w
}

func (c *ChangesWidget) Focusable() bool { return true }

func (c *ChangesWidget) SetDirs(dirs []string) {
	c.Dirs = dirs
	c.multiRoot = len(dirs) > 1
	c.Refresh()
}

func (c *ChangesWidget) Refresh() {
	oldInputs := make(map[string]*InputWidget)
	for _, g := range c.Groups {
		if g.Input != nil {
			oldInputs[g.Dir] = g.Input
		}
	}
	c.Groups = nil
	for _, dir := range c.Dirs {
		files, err := git.StatusFiles(dir)
		if err != nil {
			files = nil
		}
		var staged, unstaged []git.FileStatus
		for _, f := range files {
			if f.Staged {
				staged = append(staged, f)
			} else {
				unstaged = append(unstaged, f)
			}
		}
		input := oldInputs[dir]
		if input == nil {
			input = NewInputWidget(" > ")
		}
		c.Groups = append(c.Groups, ChangesGroup{
			Dir:             dir,
			Name:            filepath.Base(dir),
			Staged:          staged,
			Unstaged:        unstaged,
			Expanded:        !c.multiRoot,
			StagedExpanded:  true,
			ChangesExpanded: true,
			Input:           input,
		})
	}
	c.buildItems()
	c.ClampSelected(len(c.items))
}

func (c *ChangesWidget) buildItems() {
	c.items = nil
	for gi, g := range c.Groups {
		if c.multiRoot {
			c.items = append(c.items, changesItem{kind: itemHeader, groupIndex: gi})
		}
		if !c.multiRoot || g.Expanded {
			c.items = append(c.items, changesItem{kind: itemInput, groupIndex: gi})
			c.items = append(c.items, changesItem{kind: itemBorder, groupIndex: gi})
			if len(g.Staged) > 0 {
				c.items = append(c.items, changesItem{kind: itemSection, groupIndex: gi, staged: true})
				if g.StagedExpanded {
					for fi := range g.Staged {
						c.items = append(c.items, changesItem{kind: itemFile, groupIndex: gi, fileIndex: fi, staged: true})
					}
				}
			}
			if len(g.Unstaged) > 0 {
				c.items = append(c.items, changesItem{kind: itemSection, groupIndex: gi, staged: false})
				if g.ChangesExpanded {
					for fi := range g.Unstaged {
						c.items = append(c.items, changesItem{kind: itemFile, groupIndex: gi, fileIndex: fi, staged: false})
					}
				}
			}
		}
	}
}

func (c *ChangesWidget) commitGroup(gi int) {
	if gi < 0 || gi >= len(c.Groups) {
		return
	}
	g := c.Groups[gi]
	msg := g.Input.Text
	if msg == "" || len(g.Staged) == 0 {
		return
	}
	if c.OnCommit != nil {
		c.OnCommit(g.Dir, msg)
	}
}

func (c *ChangesWidget) TotalChanges() int {
	n := 0
	for _, g := range c.Groups {
		n += len(g.Staged) + len(g.Unstaged)
	}
	return n
}

func (c *ChangesWidget) SelectedFile() (dir string, status git.FileStatus, ok bool) {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return
	}
	item := c.items[c.Selected]
	if item.kind != itemFile {
		return
	}
	g := c.Groups[item.groupIndex]
	if item.staged {
		return g.Dir, g.Staged[item.fileIndex], true
	}
	return g.Dir, g.Unstaged[item.fileIndex], true
}

func (c *ChangesWidget) SelectedFullPath() string {
	dir, status, ok := c.SelectedFile()
	if !ok {
		return ""
	}
	return filepath.Join(dir, status.Path)
}

func (c *ChangesWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	if c.TotalChanges() == 0 {
		msg := "No changes"
		for i, ch := range msg {
			if i+1 < w {
				surface.SetCell(i+1, 0, term.Cell{Ch: ch, Style: term.StyleDefault})
			}
		}
		return
	}

	if h <= 0 {
		return
	}
	c.EnsureVisible(h)

	for i := 0; i < h; i++ {
		idx := c.ScrollTop + i
		if idx >= len(c.items) {
			break
		}
		item := c.items[idx]
		y := i

		style := term.StyleDefault
		if idx == c.Selected && !c.inputFocused {
			style = term.StyleSidebarSelected
		}

		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		switch item.kind {
		case itemHeader:
			c.renderHeader(surface, y, w, style, item.groupIndex)
		case itemInput:
			c.Groups[item.groupIndex].Input.Render(surface, 0, y, w)
		case itemBorder:
			for x := 0; x < w; x++ {
				surface.SetCell(x, y, term.Cell{Ch: '─', Style: term.StyleBorder})
			}
		case itemSection:
			c.renderSectionHeader(surface, y, w, style, item)
		case itemFile:
			c.renderFile(surface, y, w, style, idx == c.Selected && !c.inputFocused, item)
		}
	}
}

func (c *ChangesWidget) renderHeader(surface *RenderSurface, y, w int, style term.Style, gi int) {
	g := c.Groups[gi]
	x := 0
	chevron := '▶'
	if g.Expanded {
		chevron = '▼'
	}
	if x < w {
		surface.SetCell(x, y, term.Cell{Ch: chevron, Style: style})
		x++
	}
	if x < w {
		surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		x++
	}
	for _, ch := range g.Name {
		if x >= w {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
		x++
	}
}

func (c *ChangesWidget) renderSectionHeader(surface *RenderSurface, y, w int, style term.Style, item changesItem) {
	g := c.Groups[item.groupIndex]
	label := "Changes"
	count := len(g.Unstaged)
	expanded := g.ChangesExpanded
	if item.staged {
		label = "Staged"
		count = len(g.Staged)
		expanded = g.StagedExpanded
	}

	x := 1
	if c.multiRoot {
		x = 3
	}

	chevron := '▶'
	if expanded {
		chevron = '▼'
	}
	labelStyle := term.StyleMuted
	if style == term.StyleSidebarSelected {
		labelStyle = style
	}

	if x < w {
		surface.SetCell(x, y, term.Cell{Ch: chevron, Style: labelStyle})
		x += 2
	}

	label = fmt.Sprintf("%s (%d)", label, count)
	for _, rch := range label {
		if x >= w {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: rch, Style: labelStyle})
		x++
	}

	var actionCh rune
	if item.staged {
		actionCh = '−'
	} else {
		actionCh = '+'
	}
	if w >= 3 {
		surface.SetCell(w-2, y, term.Cell{Ch: actionCh, Style: labelStyle})
	}
}

func (c *ChangesWidget) renderFile(surface *RenderSurface, y, w int, style term.Style, selected bool, item changesItem) {
	g := c.Groups[item.groupIndex]
	var f git.FileStatus
	if item.staged {
		f = g.Staged[item.fileIndex]
	} else {
		f = g.Unstaged[item.fileIndex]
	}

	x := 1
	if c.multiRoot {
		x = 3
	}

	badge := statusBadge(f.Status)
	badgeStyle := statusStyle(f.Status)
	if selected {
		badgeStyle = style
	}
	for _, ch := range badge {
		if x < w {
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: badgeStyle})
			x++
		}
	}
	if x < w {
		surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		x++
	}

	for _, ch := range f.Path {
		if x >= w {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
		x++
	}
}

func statusStyle(status string) term.Style {
	switch status {
	case "M":
		return term.StyleDiffModified
	case "A", "?", "R":
		return term.StyleDiffAdded
	case "D":
		return term.StyleDiffDeleted
	default:
		return term.StyleDefault
	}
}

func statusBadge(status string) string {
	switch status {
	case "M":
		return "M"
	case "A":
		return "A"
	case "D":
		return "D"
	case "R":
		return "R"
	case "?":
		return "U"
	default:
		return status
	}
}

func (c *ChangesWidget) CursorPosition() (int, int, bool) {
	if !c.inputFocused {
		return 0, 0, false
	}
	r := c.GetRect()
	for i, item := range c.items {
		if item.kind == itemInput && i == c.Selected {
			inp := c.Groups[item.groupIndex].Input
			y := r.Y + i - c.ScrollTop
			return inp.CursorX(r.X), y, true
		}
	}
	return 0, 0, false
}

func (c *ChangesWidget) HandleEvent(ev tcell.Event) EventResult {
	if c.inputFocused {
		if tev, ok := ev.(*tcell.EventKey); ok {
			switch tev.Key() {
			case tcell.KeyEscape:
				c.inputFocused = false
				return EventConsumed
			case tcell.KeyEnter:
				item := c.items[c.Selected]
				c.commitGroup(item.groupIndex)
				return EventConsumed
			case tcell.KeyUp:
				c.inputFocused = false
				if c.Selected > 0 {
					c.Selected--
				}
				return EventConsumed
			case tcell.KeyDown:
				c.inputFocused = false
				if c.Selected < len(c.items)-1 {
					c.Selected++
				}
				return EventConsumed
			default:
				item := c.items[c.Selected]
				c.Groups[item.groupIndex].Input.HandleEvent(ev)
				return EventConsumed
			}
		}
	}

	r := c.GetRect()

	if tev, ok := ev.(*tcell.EventMouse); ok {
		if tev.Buttons()&tcell.Button1 != 0 {
			mx, my := tev.Position()
			idx := c.ScrollTop + (my - r.Y)
			if idx >= 0 && idx < len(c.items) {
				item := c.items[idx]
				if item.kind == itemSection && mx >= r.X+r.W-3 {
					c.Selected = idx
					g := &c.Groups[item.groupIndex]
					if item.staged {
						for _, f := range g.Staged {
							git.Unstage(g.Dir, f.Path)
						}
					} else {
						for _, f := range g.Unstaged {
							git.Stage(g.Dir, f.Path)
						}
					}
					c.Refresh()
					return EventConsumed
				}
			}
		}
	}

	res := c.SelectableList.HandleListEvent(ev, r, len(c.items))
	if res.Result == EventConsumed {
		switch res.Action {
		case ListActionActivate:
			c.activateSelected()
		case ListActionContext:
			dir, status, ok := c.SelectedFile()
			if ok && c.OnRightClick != nil {
				c.OnRightClick(dir, status, res.ScreenX, res.ScreenY)
			}
		}
		return EventConsumed
	}

	if tev, ok := ev.(*tcell.EventKey); ok {
		switch {
		case tev.Key() == tcell.KeyRune && (tev.Rune() == 'r' || tev.Rune() == 'R'):
			c.Refresh()
			return EventConsumed
		case tev.Key() == tcell.KeyRune && tev.Rune() == ' ':
			c.toggleStageSelected()
			return EventConsumed
		case tev.Key() == tcell.KeyRune && (tev.Rune() == 'a' || tev.Rune() == 'A'):
			c.stageAll()
			return EventConsumed
		case tev.Key() == tcell.KeyRune && (tev.Rune() == 'u' || tev.Rune() == 'U'):
			c.unstageAll()
			return EventConsumed
		}
	}
	return EventIgnored
}

func (c *ChangesWidget) toggleStageSelected() {
	dir, status, ok := c.SelectedFile()
	if !ok {
		return
	}
	if status.Staged {
		git.Unstage(dir, status.Path)
	} else {
		git.Stage(dir, status.Path)
	}
	c.Refresh()
}

func (c *ChangesWidget) unstageAll() {
	for _, g := range c.Groups {
		for _, f := range g.Staged {
			git.Unstage(g.Dir, f.Path)
		}
	}
	c.Refresh()
}

func (c *ChangesWidget) stageAll() {
	for _, g := range c.Groups {
		for _, f := range g.Unstaged {
			git.Stage(g.Dir, f.Path)
		}
	}
	c.Refresh()
}

func (c *ChangesWidget) activateSelected() {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return
	}
	item := c.items[c.Selected]
	switch item.kind {
	case itemHeader:
		c.Groups[item.groupIndex].Expanded = !c.Groups[item.groupIndex].Expanded
		c.buildItems()
	case itemInput:
		c.inputFocused = true
	case itemSection:
		g := &c.Groups[item.groupIndex]
		if item.staged {
			g.StagedExpanded = !g.StagedExpanded
		} else {
			g.ChangesExpanded = !g.ChangesExpanded
		}
		c.buildItems()
	case itemFile:
		dir, status, ok := c.SelectedFile()
		if ok && c.OnOpenDiff != nil {
			c.OnOpenDiff(dir, status)
		}
	}
}
