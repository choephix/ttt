package ui

import (
	"path/filepath"
	"ttt/internal/git"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ChangesGroup struct {
	Dir      string
	Name     string
	Files    []git.FileStatus
	Expanded bool
}

type changesItem struct {
	isHeader   bool
	groupIndex int
	fileIndex  int
}

type ChangesWidget struct {
	BaseWidget
	SelectableList
	Dirs         []string
	Groups       []ChangesGroup
	items        []changesItem
	multiRoot    bool
	OnOpenDiff   func(dir string, status git.FileStatus)
	OnRightClick func(dir string, status git.FileStatus, screenX, screenY int)
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
	c.Groups = nil
	for _, dir := range c.Dirs {
		files, err := git.StatusFiles(dir)
		if err != nil {
			files = nil
		}
		c.Groups = append(c.Groups, ChangesGroup{
			Dir:      dir,
			Name:     filepath.Base(dir),
			Files:    files,
			Expanded: !c.multiRoot,
		})
	}
	c.buildItems()
	c.ClampSelected(len(c.items))
}

func (c *ChangesWidget) buildItems() {
	c.items = nil
	for gi, g := range c.Groups {
		if c.multiRoot {
			c.items = append(c.items, changesItem{isHeader: true, groupIndex: gi})
		}
		if !c.multiRoot || g.Expanded {
			for fi := range g.Files {
				c.items = append(c.items, changesItem{groupIndex: gi, fileIndex: fi})
			}
		}
	}
}

func (c *ChangesWidget) totalChanges() int {
	n := 0
	for _, g := range c.Groups {
		n += len(g.Files)
	}
	return n
}

func (c *ChangesWidget) SelectedFile() (dir string, status git.FileStatus, ok bool) {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return
	}
	item := c.items[c.Selected]
	if item.isHeader {
		return
	}
	g := c.Groups[item.groupIndex]
	return g.Dir, g.Files[item.fileIndex], true
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

	if c.totalChanges() == 0 {
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
		if idx == c.Selected {
			style = term.StyleSidebarSelected
		}

		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		if item.isHeader {
			c.renderHeader(surface, y, w, style, item.groupIndex)
		} else {
			c.renderFile(surface, y, w, style, idx == c.Selected, item)
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

func (c *ChangesWidget) renderFile(surface *RenderSurface, y, w int, style term.Style, selected bool, item changesItem) {
	g := c.Groups[item.groupIndex]
	f := g.Files[item.fileIndex]

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
	case "A", "??":
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
	case "??":
		return "U"
	default:
		return status
	}
}

func (c *ChangesWidget) HandleEvent(ev tcell.Event) EventResult {
	r := c.GetRect()
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
		if tev.Key() == tcell.KeyRune && (tev.Rune() == 'r' || tev.Rune() == 'R') {
			c.Refresh()
			return EventConsumed
		}
	}
	return EventIgnored
}

func (c *ChangesWidget) activateSelected() {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return
	}
	item := c.items[c.Selected]
	if item.isHeader {
		c.Groups[item.groupIndex].Expanded = !c.Groups[item.groupIndex].Expanded
		c.buildItems()
		return
	}
	dir, status, ok := c.SelectedFile()
	if ok && c.OnOpenDiff != nil {
		c.OnOpenDiff(dir, status)
	}
}
