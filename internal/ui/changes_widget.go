package ui

import (
	"fmt"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/term"
	"path/filepath"

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
	IsPR            bool
	PRURL           string
	PRDiffs         map[string]string
}

type changesItemKind int

const (
	itemFile changesItemKind = iota
	itemHeader
	itemInput
	itemBorder
	itemSection
	itemSpacer
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
	Dirs             []string
	Groups           []ChangesGroup
	items            []changesItem
	multiRoot        bool
	inputFocused     bool
	Loading          bool
	OnOpenDiff       func(dir string, status git.FileStatus)
	OnOpenPRDiff     func(group *ChangesGroup, status git.FileStatus)
	OnRightClick     func(dir string, status git.FileStatus, screenX, screenY int)
	OnCommit         func(dir string, message string)
	OnGroupMenu      func(dir string, screenX, screenY int)
	OnPRGroupMenu    func(group *ChangesGroup, screenX, screenY int)
	OnConfirmDiscard func(message string, onConfirm func())
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
	oldGroups := make(map[string]ChangesGroup)
	var prGroups []ChangesGroup
	for _, g := range c.Groups {
		oldGroups[g.Dir] = g
		if g.IsPR {
			prGroups = append(prGroups, g)
		}
	}
	c.Groups = nil
	// multiple workspace folders may resolve to the same git root
	seen := make(map[string]bool)
	for _, dir := range c.Dirs {
		if root := git.RepoRoot(dir); root != "" {
			dir = root
		}
		if seen[dir] {
			continue
		}
		seen[dir] = true
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
		input := NewInputWidget()
		input.Placeholder = "Message"
		expanded := !c.multiRoot
		stagedExpanded := true
		changesExpanded := true
		if old, ok := oldGroups[dir]; ok {
			if old.Input != nil {
				input = old.Input
			}
			expanded = old.Expanded
			stagedExpanded = old.StagedExpanded
			changesExpanded = old.ChangesExpanded
		}
		c.Groups = append(c.Groups, ChangesGroup{
			Dir:             dir,
			Name:            filepath.Base(dir),
			Staged:          staged,
			Unstaged:        unstaged,
			Expanded:        expanded,
			StagedExpanded:  stagedExpanded,
			ChangesExpanded: changesExpanded,
			Input:           input,
		})
	}
	c.Groups = append(c.Groups, prGroups...)
	c.multiRoot = len(c.Groups) > 1
	c.buildItems()
	c.ClampSelected(len(c.items))
}

func (c *ChangesWidget) buildItems() {
	c.items = nil
	for gi, g := range c.Groups {
		showHeader := c.multiRoot || g.IsPR
		if showHeader {
			c.items = append(c.items, changesItem{kind: itemHeader, groupIndex: gi})
		}
		if !showHeader || g.Expanded {
			if showHeader {
				c.items = append(c.items, changesItem{kind: itemBorder, groupIndex: gi})
			}
			if !g.IsPR {
				c.items = append(c.items, changesItem{kind: itemInput, groupIndex: gi})
				c.items = append(c.items, changesItem{kind: itemBorder, groupIndex: gi})
			}
			if !g.IsPR && len(g.Staged) > 0 {
				c.items = append(c.items, changesItem{kind: itemSection, groupIndex: gi, staged: true})
				if g.StagedExpanded {
					for fi := range g.Staged {
						c.items = append(c.items, changesItem{kind: itemFile, groupIndex: gi, fileIndex: fi, staged: true})
					}
				}
			}
			if len(g.Unstaged) > 0 {
				if !g.IsPR {
					c.items = append(c.items, changesItem{kind: itemSection, groupIndex: gi, staged: false})
				}
				if g.ChangesExpanded {
					for fi := range g.Unstaged {
						c.items = append(c.items, changesItem{kind: itemFile, groupIndex: gi, fileIndex: fi, staged: false})
					}
				}
			}
			c.items = append(c.items, changesItem{kind: itemBorder, groupIndex: gi})
			c.items = append(c.items, changesItem{kind: itemSpacer, groupIndex: gi})
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
		if c.Loading {
			msg = "Loading..."
		}
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

		indent := 0

		switch item.kind {
		case itemHeader:
			c.renderHeader(surface, y, w, style, item.groupIndex)
		case itemInput:
			c.Groups[item.groupIndex].Input.Render(surface, indent, y, w-indent)
		case itemBorder:
			for x := indent; x < w; x++ {
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
	maxNameX := w - 3
	for _, ch := range g.Name {
		if x >= maxNameX {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
		x++
	}
	if w >= 3 {
		surface.SetCell(w-2, y, term.Cell{Ch: '⋮', Style: style})
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

	if w >= 5 {
		if item.staged {
			surface.SetCell(w-2, y, term.Cell{Ch: '−', Style: labelStyle})
		} else {
			surface.SetCell(w-4, y, term.Cell{Ch: '✕', Style: labelStyle})
			surface.SetCell(w-2, y, term.Cell{Ch: '+', Style: labelStyle})
		}
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

	isPR := c.Groups[item.groupIndex].IsPR
	maxPathX := w - 4
	if isPR {
		maxPathX = w - 1
	}
	for _, ch := range f.Path {
		if x >= maxPathX {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
		x++
	}

	if !isPR {
		if item.staged {
			if w >= 3 {
				surface.SetCell(w-2, y, term.Cell{Ch: '−', Style: style})
			}
		} else {
			if w >= 3 {
				surface.SetCell(w-2, y, term.Cell{Ch: '+', Style: style})
			}
		}
	}
}

func statusStyle(status string) term.Style {
	switch status {
	case "M":
		return term.StyleWarning
	case "A", "?", "R":
		return term.StyleSuccess
	case "D":
		return term.StyleDanger
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

func (c *ChangesWidget) FocusedInput() *InputWidget {
	if !c.inputFocused {
		return nil
	}
	if c.Selected >= 0 && c.Selected < len(c.items) {
		item := c.items[c.Selected]
		if item.kind == itemInput {
			return c.Groups[item.groupIndex].Input
		}
	}
	return nil
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
				if item.kind == itemHeader && mx >= r.X+r.W-3 {
					g := &c.Groups[item.groupIndex]
					if g.IsPR {
						if c.OnPRGroupMenu != nil {
							c.OnPRGroupMenu(g, mx, my)
						}
					} else if c.OnGroupMenu != nil {
						c.OnGroupMenu(g.Dir, mx, my)
					}
					return EventConsumed
				}
				if item.kind == itemSection {
					if !item.staged && mx >= r.X+r.W-5 && mx < r.X+r.W-3 {
						c.Selected = idx
						c.handleSectionAction(item)
						return EventConsumed
					}
					if mx >= r.X+r.W-3 {
						c.Selected = idx
						if item.staged {
							c.handleSectionAction(item)
						} else {
							c.handleSectionStageAll(item)
						}
						return EventConsumed
					}
				}
				if item.kind == itemInput {
					c.Selected = idx
					c.inputFocused = true
					c.Groups[item.groupIndex].Input.HandleTextClick(mx)
					return EventConsumed
				}
				if item.kind == itemFile && mx >= r.X+r.W-3 {
					c.Selected = idx
					c.handleFileAction(item)
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
		inPR := c.selectedInPR()
		switch {
		case tev.Key() == tcell.KeyRune && (tev.Rune() == 'r' || tev.Rune() == 'R'):
			c.Refresh()
			return EventConsumed
		case !inPR && tev.Key() == tcell.KeyRune && tev.Rune() == ' ':
			c.toggleStageSelected()
			return EventConsumed
		case !inPR && tev.Key() == tcell.KeyRune && (tev.Rune() == 'a' || tev.Rune() == 'A'):
			c.stageAll()
			return EventConsumed
		case !inPR && tev.Key() == tcell.KeyRune && (tev.Rune() == 'u' || tev.Rune() == 'U'):
			c.unstageAll()
			return EventConsumed
		case !inPR && tev.Key() == tcell.KeyRune && tev.Rune() == 'd':
			c.discardSelected()
			return EventConsumed
		case !inPR && tev.Key() == tcell.KeyRune && tev.Rune() == 'D':
			c.discardAllInGroup()
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

func (c *ChangesWidget) discardSelected() {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return
	}
	item := c.items[c.Selected]
	if item.kind != itemFile || item.staged {
		return
	}
	g := c.Groups[item.groupIndex]
	f := g.Unstaged[item.fileIndex]
	c.confirmDiscard(g.Dir, f)
}

func (c *ChangesWidget) discardAllInGroup() {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return
	}
	item := c.items[c.Selected]
	gi := item.groupIndex
	if len(c.Groups[gi].Unstaged) == 0 {
		return
	}
	c.confirmDiscardAll(gi)
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
		g := &c.Groups[item.groupIndex]
		if g.IsPR {
			_, status, ok := c.SelectedFile()
			if ok && c.OnOpenPRDiff != nil {
				c.OnOpenPRDiff(g, status)
			}
		} else {
			dir, status, ok := c.SelectedFile()
			if ok && c.OnOpenDiff != nil {
				c.OnOpenDiff(dir, status)
			}
		}
	}
}

func (c *ChangesWidget) handleSectionAction(item changesItem) {
	g := &c.Groups[item.groupIndex]
	if item.staged {
		for _, f := range g.Staged {
			git.Unstage(g.Dir, f.Path)
		}
		c.Refresh()
	} else {
		c.confirmDiscardAll(item.groupIndex)
	}
}

func (c *ChangesWidget) handleSectionStageAll(item changesItem) {
	g := &c.Groups[item.groupIndex]
	for _, f := range g.Unstaged {
		git.Stage(g.Dir, f.Path)
	}
	c.Refresh()
}

func (c *ChangesWidget) handleFileAction(item changesItem) {
	g := c.Groups[item.groupIndex]
	if item.staged {
		f := g.Staged[item.fileIndex]
		git.Unstage(g.Dir, f.Path)
	} else {
		f := g.Unstaged[item.fileIndex]
		git.Stage(g.Dir, f.Path)
	}
	c.Refresh()
}

func (c *ChangesWidget) confirmDiscard(dir string, f git.FileStatus) {
	if c.OnConfirmDiscard == nil {
		return
	}
	msg := fmt.Sprintf("Discard changes to %s? This is irreversible.", f.Path)
	if f.Status == "?" {
		msg = fmt.Sprintf("Delete untracked file %s? This is irreversible.", f.Path)
	}
	c.OnConfirmDiscard(msg, func() {
		if f.Status == "?" {
			git.DiscardUntracked(dir, f.Path)
		} else {
			git.Discard(dir, f.Path)
		}
		c.Refresh()
	})
}

func (c *ChangesWidget) confirmDiscardAll(gi int) {
	if c.OnConfirmDiscard == nil {
		return
	}
	g := c.Groups[gi]
	msg := fmt.Sprintf("Discard all %d changes? This is irreversible.", len(g.Unstaged))
	c.OnConfirmDiscard(msg, func() {
		for _, f := range g.Unstaged {
			if f.Status == "?" {
				git.DiscardUntracked(g.Dir, f.Path)
			} else {
				git.Discard(g.Dir, f.Path)
			}
		}
		c.Refresh()
	})
}

func (c *ChangesWidget) selectedInPR() bool {
	if c.Selected < 0 || c.Selected >= len(c.items) {
		return false
	}
	return c.Groups[c.items[c.Selected].groupIndex].IsPR
}

func (c *ChangesWidget) AddPRGroup(name, url string, files []git.FileStatus, diffs map[string]string) {
	c.Groups = append(c.Groups, ChangesGroup{
		Dir:             "pr://" + name,
		Name:            name,
		Unstaged:        files,
		Expanded:        true,
		ChangesExpanded: true,
		IsPR:            true,
		PRURL:           url,
		PRDiffs:         diffs,
	})
	c.multiRoot = len(c.Groups) > 1
	c.buildItems()
	c.ClampSelected(len(c.items))
}

func (c *ChangesWidget) RemovePRGroup(name string) {
	var kept []ChangesGroup
	for _, g := range c.Groups {
		if !(g.IsPR && g.Name == name) {
			kept = append(kept, g)
		}
	}
	c.Groups = kept
	c.multiRoot = len(c.Groups) > 1
	c.buildItems()
	c.ClampSelected(len(c.items))
}

func (c *ChangesWidget) RemovePRGroups() {
	var kept []ChangesGroup
	for _, g := range c.Groups {
		if !g.IsPR {
			kept = append(kept, g)
		}
	}
	c.Groups = kept
	c.multiRoot = len(c.Groups) > 1
	c.buildItems()
	c.ClampSelected(len(c.items))
}
