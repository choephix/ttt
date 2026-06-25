package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type ChangesPanel struct {
	Tree    *widgets.TreeWidget
	Input   *widgets.InputWidget
	Adapter *ui.WidgetAdapter
	Dirs    []string

	groups    []changesGroup
	multiRoot bool
	expanded  map[string]bool

	OnOpenDiff       func(dir string, status git.FileStatus, extended bool)
	OnOpenPRDiff     func(group *ui.ChangesGroup, status git.FileStatus, extended bool)
	OnOpenFile       func(path string)
	OnRightClick     func(dir string, status git.FileStatus, screenX, screenY int)
	OnCommit         func(dir string, message string)
	OnGroupMenu      func(dir string, screenX, screenY int)
	OnPRGroupMenu    func(group *ui.ChangesGroup, screenX, screenY int)
	OnRefreshPR      func(url string)
	OnConfirmDiscard func(message string, onConfirm func())

	PRGroups []prGroup
}

type changesGroup struct {
	Dir      string
	Name     string
	Staged   []git.FileStatus
	Unstaged []git.FileStatus
}

type prGroup struct {
	Dir       string
	Name      string
	Files     []git.FileStatus
	PRURL     string
	PRDiffs   map[string]string
	PROwner   string
	PRRepo    string
	PRBaseSHA string
	PRHeadSHA string
}

func NewChangesPanel(dirs ...string) *ChangesPanel {
	cp := &ChangesPanel{
		Dirs:      dirs,
		multiRoot: len(dirs) > 1,
		expanded:  make(map[string]bool),
	}

	cp.Input = widgets.NewInputWidget(widgets.InputConfig{
		Placeholder: "Message",
		Bordered:    false,
		OnSubmit: func(text string) {
			cp.commitFocusedGroup()
		},
	})

	cp.Tree = widgets.NewTreeWidget(widgets.TreeConfig{
		Indent:    1,
		EmptyText: "No changes",
		OnCommand: func(cmd string, node *widgets.TreeNode) {
			cp.handleCommand(cmd, node)
		},
		OnMenu: func(_ []widgets.MenuEntry, node *widgets.TreeNode, sx, sy int) {
			cp.handleMenu(node, sx, sy)
		},
		OnKey: func(ev *tcell.EventKey, node *widgets.TreeNode) bool {
			return cp.handleKey(ev)
		},
	})

	divTop := widgets.NewDividerWidget(widgets.DividerConfig{})
	divBottom := widgets.NewDividerWidget(widgets.DividerConfig{})

	vstack := widgets.NewVStackWidget(cp.Input, divTop, cp.Tree, divBottom)

	cp.Adapter = ui.NewWidgetAdapter(vstack)

	cp.Refresh()
	return cp
}

func (cp *ChangesPanel) SetDirs(dirs []string) {
	cp.Dirs = dirs
	cp.multiRoot = len(dirs) > 1
	cp.Refresh()
}

func (cp *ChangesPanel) Refresh() {
	cp.saveExpanded()
	cp.groups = nil
	seen := make(map[string]bool)
	for _, dir := range cp.Dirs {
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
		cp.groups = append(cp.groups, changesGroup{
			Dir:      dir,
			Name:     filepath.Base(dir),
			Staged:   staged,
			Unstaged: unstaged,
		})
	}
	cp.multiRoot = len(cp.groups)+len(cp.PRGroups) > 1
	cp.buildTree()
}

func (cp *ChangesPanel) saveExpanded() {
	for _, node := range cp.Tree.FlatList() {
		if node.Expandable || len(node.Children) > 0 {
			cp.expanded[node.ID] = node.Expanded
		}
	}
}

func (cp *ChangesPanel) restoreExpanded(node *widgets.TreeNode) {
	if exp, ok := cp.expanded[node.ID]; ok {
		node.Expanded = exp
	}
	for _, child := range node.Children {
		cp.restoreExpanded(child)
	}
}

func (cp *ChangesPanel) buildTree() {
	var roots []*widgets.TreeNode

	for gi, g := range cp.groups {
		var sectionNodes []*widgets.TreeNode

		if len(g.Staged) > 0 {
			stagedNode := &widgets.TreeNode{
				ID:         fmt.Sprintf("staged:%d", gi),
				Label:      fmt.Sprintf("Staged (%d)", len(g.Staged)),
				Expandable: true,
				Expanded:   true,
				Muted:      true,
				Actions: []widgets.Action{
					{Icon: "−", Command: "unstageAll"},
				},
			}
			for _, f := range g.Staged {
				child := cp.fileNode(g.Dir, f, true)
				stagedNode.Children = append(stagedNode.Children, child)
			}
			sectionNodes = append(sectionNodes, stagedNode)
		}

		if len(g.Unstaged) > 0 {
			changesNode := &widgets.TreeNode{
				ID:         fmt.Sprintf("changes:%d", gi),
				Label:      fmt.Sprintf("Changes (%d)", len(g.Unstaged)),
				Expandable: true,
				Expanded:   true,
				Muted:      true,
				Actions: []widgets.Action{
					{Icon: "✕", Command: "discardAll"},
					{Icon: "+", Command: "stageAll"},
				},
			}
			for _, f := range g.Unstaged {
				child := cp.fileNode(g.Dir, f, false)
				changesNode.Children = append(changesNode.Children, child)
			}
			sectionNodes = append(sectionNodes, changesNode)
		}

		if cp.multiRoot {
			root := &widgets.TreeNode{
				ID:         fmt.Sprintf("root:%d", gi),
				Label:      g.Name,
				Expandable: true,
				Expanded:   true,
				Children:   sectionNodes,
				Actions: []widgets.Action{
					{Icon: "⋮", Command: "groupMenu"},
				},
			}
			roots = append(roots, root)
		} else {
			roots = append(roots, sectionNodes...)
		}
	}

	for pi, pg := range cp.PRGroups {
		prRoot := &widgets.TreeNode{
			ID:         fmt.Sprintf("pr:%d", pi),
			Label:      pg.Name,
			Expandable: true,
			Expanded:   true,
			Actions: []widgets.Action{
				{Icon: "⋮", Command: "prGroupMenu"},
			},
		}
		for _, f := range pg.Files {
			child := cp.fileNode(pg.Dir, f, false)
			child.Actions = nil
			prRoot.Children = append(prRoot.Children, child)
		}
		roots = append(roots, prRoot)
	}

	for _, root := range roots {
		cp.restoreExpanded(root)
	}

	cp.Tree.SetItems(roots)
}

func (cp *ChangesPanel) fileNode(dir string, f git.FileStatus, staged bool) *widgets.TreeNode {
	icon := ui.StatusBadge(f.Status)
	iconStyle := ui.StatusStyle(f.Status)
	actionIcon := "+"
	actionCmd := "stage"
	if staged {
		actionIcon = "−"
		actionCmd = "unstage"
	}
	return &widgets.TreeNode{
		ID:        fmt.Sprintf("file:%s:%s:%v", dir, f.Path, staged),
		Label:     f.Path,
		Icon:      icon,
		IconStyle: iconStyle,
		Actions: []widgets.Action{
			{Icon: actionIcon, Command: actionCmd},
		},
	}
}

func (cp *ChangesPanel) TotalChanges() int {
	n := 0
	for _, g := range cp.groups {
		n += len(g.Staged) + len(g.Unstaged)
	}
	return n
}

func (cp *ChangesPanel) commitFocusedGroup() {
	msg := cp.Input.Text()
	if msg == "" {
		return
	}
	dir := cp.selectedGroupDir()
	if dir == "" {
		for _, g := range cp.groups {
			if len(g.Staged) > 0 {
				dir = g.Dir
				break
			}
		}
	}
	if dir != "" && cp.OnCommit != nil {
		cp.OnCommit(dir, msg)
		cp.Input.Clear()
	}
}

func (cp *ChangesPanel) selectedGroupDir() string {
	node := cp.Tree.Selected()
	if node == nil {
		return ""
	}
	dir, _, _, ok := cp.parseFileNode(node)
	if ok {
		return dir
	}
	gi := cp.groupIndexFromNode(node)
	if gi >= 0 && gi < len(cp.groups) {
		return cp.groups[gi].Dir
	}
	if strings.HasPrefix(node.ID, "root:") {
		var idx int
		if _, err := fmt.Sscanf(node.ID, "root:%d", &idx); err == nil && idx < len(cp.groups) {
			return cp.groups[idx].Dir
		}
	}
	return ""
}

func (cp *ChangesPanel) selectedInPR() bool {
	node := cp.Tree.Selected()
	if node == nil {
		return false
	}
	dir, _, _, ok := cp.parseFileNode(node)
	if ok {
		for _, pg := range cp.PRGroups {
			if pg.Dir == dir {
				return true
			}
		}
		return false
	}
	return strings.HasPrefix(node.ID, "pr:")
}

func (cp *ChangesPanel) handleCommand(cmd string, node *widgets.TreeNode) {
	dir, status, staged, ok := cp.parseFileNode(node)
	switch cmd {
	case "activate":
		if ok {
			cp.openDiff(dir, status, staged, false)
		}
	case "stage":
		if ok && !staged {
			git.Stage(dir, status.Path)
			cp.Refresh()
		}
	case "unstage":
		if ok && staged {
			git.Unstage(dir, status.Path)
			cp.Refresh()
		}
	case "stageAll":
		cp.stageAll()
	case "unstageAll":
		cp.unstageAll()
	case "discardAll":
		gi := cp.groupIndexFromNode(node)
		if gi >= 0 {
			cp.confirmDiscardAll(gi)
		}
	case "groupMenu":
		r := cp.Tree.GetRect()
		cp.handleMenu(node, r.X+r.W-2, r.Y+cp.Tree.SelectedIndex()-cp.Tree.ScrollTop())
	case "prGroupMenu":
		r := cp.Tree.GetRect()
		cp.handleMenu(node, r.X+r.W-2, r.Y+cp.Tree.SelectedIndex()-cp.Tree.ScrollTop())
	}
}

func (cp *ChangesPanel) handleKey(ev *tcell.EventKey) bool {
	if ev.Key() != tcell.KeyRune {
		return false
	}
	inPR := cp.selectedInPR()
	switch ev.Rune() {
	case 'r', 'R':
		if inPR {
			cp.refreshSelectedPR()
		} else {
			cp.Refresh()
		}
		return true
	case ' ':
		if !inPR {
			cp.ToggleStageSelected()
		}
		return true
	case 'a', 'A':
		if !inPR {
			cp.stageAll()
		}
		return true
	case 'u', 'U':
		if !inPR {
			cp.unstageAll()
		}
		return true
	case 'd':
		if !inPR {
			cp.DiscardSelected()
		}
		return true
	case 'D':
		if !inPR {
			node := cp.Tree.Selected()
			if node != nil {
				gi := cp.groupIndexFromNode(node)
				if gi >= 0 {
					cp.confirmDiscardAll(gi)
				}
			}
		}
		return true
	case 'o', 'v':
		cp.OpenSelectedFile()
		return true
	case 'c':
		cp.OpenSelectedDiff(false)
		return true
	case 'e':
		cp.OpenSelectedDiff(true)
		return true
	}
	return false
}

func (cp *ChangesPanel) refreshSelectedPR() {
	node := cp.Tree.Selected()
	if node == nil {
		return
	}
	for _, pg := range cp.PRGroups {
		if strings.HasPrefix(node.ID, "pr:") && node.Label == pg.Name {
			if cp.OnRefreshPR != nil && pg.PRURL != "" {
				cp.RemovePRGroup(pg.Name)
				cp.OnRefreshPR(pg.PRURL)
			}
			return
		}
	}
	dir, _, _, ok := cp.parseFileNode(node)
	if ok {
		for _, pg := range cp.PRGroups {
			if pg.Dir == dir {
				if cp.OnRefreshPR != nil && pg.PRURL != "" {
					cp.RemovePRGroup(pg.Name)
					cp.OnRefreshPR(pg.PRURL)
				}
				return
			}
		}
	}
}

func (cp *ChangesPanel) handleMenu(node *widgets.TreeNode, sx, sy int) {
	dir, status, _, ok := cp.parseFileNode(node)
	if ok && cp.OnRightClick != nil {
		cp.OnRightClick(dir, status, sx, sy)
		return
	}
	for _, pg := range cp.PRGroups {
		if node.ID == fmt.Sprintf("pr:%s", pg.Name) || node.Label == pg.Name {
			if cp.OnPRGroupMenu != nil {
				uiGroup := cp.toUIChangesGroup(&pg)
				cp.OnPRGroupMenu(uiGroup, sx, sy)
			}
			return
		}
	}
	for gi, g := range cp.groups {
		if node.ID == fmt.Sprintf("root:%d", gi) {
			if cp.OnGroupMenu != nil {
				cp.OnGroupMenu(g.Dir, sx, sy)
			}
			return
		}
	}
}

func (cp *ChangesPanel) openDiff(dir string, status git.FileStatus, staged bool, extended bool) {
	for _, pg := range cp.PRGroups {
		if pg.Dir == dir {
			if cp.OnOpenPRDiff != nil {
				uiGroup := cp.toUIChangesGroup(&pg)
				cp.OnOpenPRDiff(uiGroup, status, extended)
			}
			return
		}
	}
	if cp.OnOpenDiff != nil {
		cp.OnOpenDiff(dir, status, extended)
	}
}

func (cp *ChangesPanel) parseFileNode(node *widgets.TreeNode) (dir string, status git.FileStatus, staged bool, ok bool) {
	id := node.ID
	if len(id) < 6 || id[:5] != "file:" {
		return
	}
	rest := id[5:]
	lastColon := -1
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == ':' {
			lastColon = i
			break
		}
	}
	if lastColon < 0 {
		return
	}
	s := rest[lastColon+1:] == "true"
	rest = rest[:lastColon]

	secondLastColon := -1
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == ':' {
			secondLastColon = i
			break
		}
	}
	if secondLastColon < 0 {
		return
	}
	d := rest[:secondLastColon]
	path := rest[secondLastColon+1:]

	for _, g := range cp.groups {
		if g.Dir == d {
			files := g.Unstaged
			if s {
				files = g.Staged
			}
			for _, f := range files {
				if f.Path == path {
					return d, f, s, true
				}
			}
		}
	}
	for _, pg := range cp.PRGroups {
		if pg.Dir == d {
			for _, f := range pg.Files {
				if f.Path == path {
					return d, f, false, true
				}
			}
		}
	}
	return
}

func (cp *ChangesPanel) groupIndexFromNode(node *widgets.TreeNode) int {
	var gi int
	if _, err := fmt.Sscanf(node.ID, "changes:%d", &gi); err == nil {
		return gi
	}
	if _, err := fmt.Sscanf(node.ID, "staged:%d", &gi); err == nil {
		return gi
	}
	if _, err := fmt.Sscanf(node.ID, "root:%d", &gi); err == nil {
		return gi
	}
	return -1
}

func (cp *ChangesPanel) stageAll() {
	for _, g := range cp.groups {
		for _, f := range g.Unstaged {
			git.Stage(g.Dir, f.Path)
		}
	}
	cp.Refresh()
}

func (cp *ChangesPanel) unstageAll() {
	for _, g := range cp.groups {
		for _, f := range g.Staged {
			git.Unstage(g.Dir, f.Path)
		}
	}
	cp.Refresh()
}

func (cp *ChangesPanel) confirmDiscard(dir string, f git.FileStatus) {
	if cp.OnConfirmDiscard == nil {
		return
	}
	msg := fmt.Sprintf("Discard changes to %s? This is irreversible.", f.Path)
	if f.Status == "?" {
		msg = fmt.Sprintf("Delete untracked file %s? This is irreversible.", f.Path)
	}
	cp.OnConfirmDiscard(msg, func() {
		if f.Status == "?" {
			git.DiscardUntracked(dir, f.Path)
		} else {
			git.Discard(dir, f.Path)
		}
		cp.Refresh()
	})
}

func (cp *ChangesPanel) confirmDiscardAll(gi int) {
	if cp.OnConfirmDiscard == nil || gi < 0 || gi >= len(cp.groups) {
		return
	}
	g := cp.groups[gi]
	msg := fmt.Sprintf("Discard all %d changes? This is irreversible.", len(g.Unstaged))
	cp.OnConfirmDiscard(msg, func() {
		for _, f := range g.Unstaged {
			if f.Status == "?" {
				git.DiscardUntracked(g.Dir, f.Path)
			} else {
				git.Discard(g.Dir, f.Path)
			}
		}
		cp.Refresh()
	})
}

func (cp *ChangesPanel) SelectedFile() (dir string, status git.FileStatus, ok bool) {
	node := cp.Tree.Selected()
	if node == nil {
		return
	}
	dir, status, _, ok = cp.parseFileNode(node)
	return
}

func (cp *ChangesPanel) SelectedFullPath() string {
	dir, status, ok := cp.SelectedFile()
	if !ok {
		return ""
	}
	return filepath.Join(dir, status.Path)
}

func (cp *ChangesPanel) SelectedGroup() *ui.ChangesGroup {
	node := cp.Tree.Selected()
	if node == nil {
		return nil
	}
	for _, pg := range cp.PRGroups {
		if node.Label == pg.Name {
			return cp.toUIChangesGroup(&pg)
		}
	}
	_, _, _, ok := cp.parseFileNode(node)
	if ok {
		dir, _, _ := cp.SelectedFile()
		for _, g := range cp.groups {
			if g.Dir == dir {
				return &ui.ChangesGroup{
					Dir:      g.Dir,
					Name:     g.Name,
					Staged:   g.Staged,
					Unstaged: g.Unstaged,
				}
			}
		}
	}
	return nil
}

func (cp *ChangesPanel) toUIChangesGroup(pg *prGroup) *ui.ChangesGroup {
	return &ui.ChangesGroup{
		Dir:             pg.Dir,
		Name:            pg.Name,
		Unstaged:        pg.Files,
		IsPR:            true,
		PRURL:           pg.PRURL,
		PRDiffs:         pg.PRDiffs,
		PROwner:         pg.PROwner,
		PRRepo:          pg.PRRepo,
		PRBaseSHA:       pg.PRBaseSHA,
		PRHeadSHA:       pg.PRHeadSHA,
		Expanded:        true,
		ChangesExpanded: true,
	}
}

func (cp *ChangesPanel) AddPRGroup(name, url, owner, repo, baseSHA, headSHA string, files []git.FileStatus, diffs map[string]string) {
	cp.PRGroups = append(cp.PRGroups, prGroup{
		Dir:       "pr://" + name,
		Name:      name,
		Files:     files,
		PRURL:     url,
		PRDiffs:   diffs,
		PROwner:   owner,
		PRRepo:    repo,
		PRBaseSHA: baseSHA,
		PRHeadSHA: headSHA,
	})
	cp.multiRoot = len(cp.groups)+len(cp.PRGroups) > 1
	cp.buildTree()
}

func (cp *ChangesPanel) RemovePRGroup(name string) {
	var kept []prGroup
	for _, pg := range cp.PRGroups {
		if pg.Name != name {
			kept = append(kept, pg)
		}
	}
	cp.PRGroups = kept
	cp.multiRoot = len(cp.groups)+len(cp.PRGroups) > 1
	cp.buildTree()
}

func (cp *ChangesPanel) RemovePRGroups() {
	cp.PRGroups = nil
	cp.multiRoot = len(cp.groups)+len(cp.PRGroups) > 1
	cp.buildTree()
}

func (cp *ChangesPanel) DiscardSelected() {
	dir, status, _, ok := cp.parseFileNode(cp.Tree.Selected())
	if !ok || status.Staged {
		return
	}
	cp.confirmDiscard(dir, status)
}

func (cp *ChangesPanel) ToggleStageSelected() {
	node := cp.Tree.Selected()
	if node == nil {
		return
	}
	dir, status, staged, ok := cp.parseFileNode(node)
	if !ok {
		return
	}
	if staged {
		git.Unstage(dir, status.Path)
	} else {
		git.Stage(dir, status.Path)
	}
	cp.Refresh()
}

func (cp *ChangesPanel) OpenSelectedDiff(extended bool) {
	node := cp.Tree.Selected()
	if node == nil {
		return
	}
	dir, status, staged, ok := cp.parseFileNode(node)
	if !ok {
		return
	}
	cp.openDiff(dir, status, staged, extended)
}

func (cp *ChangesPanel) OpenSelectedFile() {
	if cp.OnOpenFile != nil {
		if path := cp.SelectedFullPath(); path != "" {
			cp.OnOpenFile(path)
		}
	}
}

func (cp *ChangesPanel) Groups() []ui.ChangesGroup {
	var result []ui.ChangesGroup
	for _, g := range cp.groups {
		result = append(result, ui.ChangesGroup{
			Dir:      g.Dir,
			Name:     g.Name,
			Staged:   g.Staged,
			Unstaged: g.Unstaged,
		})
	}
	for _, pg := range cp.PRGroups {
		result = append(result, *cp.toUIChangesGroup(&pg))
	}
	return result
}

func (cp *ChangesPanel) ClearInput(dir string) {
	for _, g := range cp.groups {
		if g.Dir == dir {
			cp.Input.Clear()
			return
		}
	}
}
