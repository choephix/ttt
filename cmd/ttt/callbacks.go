package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/core/diff"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/ui"

	"github.com/gdamore/tcell/v2"
)

func (a *App) ShowSidebarMoreMenu(sx, sy int) {
	var items []ui.ContextMenuItem
	switch a.sidebar.ActivePanel {
	case "explorer":
		items = []ui.ContextMenuItem{
			{Label: "New File", Command: "file.new"},
			{Label: "Add Folder", Command: "workspace.addFolder"},
			{Label: "Refresh", Command: "explorer.refresh"},
		}
	case "search":
		replaceLabel := "Replace"
		if a.search.IsReplaceMode() {
			replaceLabel = "Search"
		}
		items = []ui.ContextMenuItem{
			{Label: replaceLabel, Shortcut: a.KeyFor("sidebar.searchReplace"), Command: "sidebar.searchReplace"},
			ui.MenuSep(),
			{Label: "Clear Results", Command: "search.clear"},
		}
	case "changes":
		items = []ui.ContextMenuItem{
			{Label: "Refresh", Command: "changes.refresh"},
			{Label: "Open Pull Request", Command: "pr.open"},
			ui.MenuSep(),
			{Label: "Pull", Command: "git.pull"},
			{Label: "Push", Command: "git.push"},
			{Label: "Sync", Command: "git.sync"},
		}
	}
	if len(items) > 0 {
		openContextMenu(a, items, sx, sy)
	}
}

func (a *App) DiffSearchSources() []ui.DiffSearchSource {
	seen := map[string]bool{}
	sources := a.editorGroup.DiffTabSources()
	for _, s := range sources {
		seen[s.TabName] = true
	}
	for _, g := range a.changes.Groups {
		if !g.IsPR {
			continue
		}
		for path, diffText := range g.PRDiffs {
			tabName := path + " (diff)"
			if seen[tabName] {
				continue
			}
			fd := diff.Parse(diffText)
			dv := ui.NewDiffViewWidget(path, fd)
			sources = append(sources, ui.DiffSearchSource{TabName: tabName, Lines: dv.CombinedLines()})
		}
	}
	return sources
}

func (a *App) NavigateToSearchMatch(path string, line, col int) {
	if strings.HasSuffix(path, " (diff)") {
		if !a.editorGroup.SwitchToTabByPath(path) {
			filePath := strings.TrimSuffix(path, " (diff)")
			for _, g := range a.changes.Groups {
				if !g.IsPR {
					continue
				}
				if diffText, ok := g.PRDiffs[filePath]; ok {
					a.editorGroup.OpenDiff(filePath, diff.Parse(diffText))
					break
				}
			}
		}
		if dv := a.editorGroup.ActiveDiffWidget(); dv != nil {
			dv.ScrollToLine(line - 1)
			dv.ApplySearchHighlight(a.search.Input.Text, a.search.Options)
		}
		a.root.SetFocus(a.editorGroup)
		return
	}
	a.editorGroup.OpenFile(path)
	a.editorGroup.GoToLine(line)
	if a.search.Input.Text != "" {
		matches, _ := ui.FindInLines(a.editorGroup.Editor.Buf.Lines, a.search.Input.Text, a.search.Options)
		a.editorGroup.SetSearch(a.search.Input.Text, matches)
	}
	a.root.SetFocus(a.editorGroup)
}

func (a *App) PreviewSearchReplace(filePath string, matches []ui.SearchMatch, replacement string, opts ui.SearchOptions) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		a.StatusWarn("Cannot read file: " + err.Error())
		return
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	fd := ui.BuildReplaceDiff(filepath.Base(filePath), lines, matches, replacement, opts)
	a.editorGroup.OpenDiff(filePath, fd)
	a.root.SetFocus(a.editorGroup)
}

func (a *App) ApplySearchReplace(filePath string, matches []ui.SearchMatch, replacement string, opts ui.SearchOptions) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		a.StatusWarn("Cannot read file: " + err.Error())
		return
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	newLines := ui.ApplyReplacements(lines, matches, replacement, opts)
	if err := os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")+"\n"), 0644); err != nil {
		a.StatusWarn("Cannot write file: " + err.Error())
		return
	}
	a.editorGroup.ReloadFile(filePath)
	a.search.Refresh()
	a.StatusNotify(fmt.Sprintf("Replaced %d matches in %s", len(matches), filepath.Base(filePath)))
}

func (a *App) ApplySearchReplaceAll(allMatches map[string][]ui.SearchMatch, replacement string, opts ui.SearchOptions) {
	totalFiles := len(allMatches)
	totalMatches := 0
	for _, m := range allMatches {
		totalMatches += len(m)
	}
	msg := fmt.Sprintf("Replace %d matches across %d files? This cannot be undone.", totalMatches, totalFiles)
	a.ShowConfirmDialog(msg, []string{"Cancel", "Replace All"}, []func(){
		func() { a.DismissDialog() },
		func() {
			a.DismissDialog()
			for filePath, matches := range allMatches {
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}
				lines := strings.Split(string(data), "\n")
				if len(lines) > 0 && lines[len(lines)-1] == "" {
					lines = lines[:len(lines)-1]
				}
				newLines := ui.ApplyReplacements(lines, matches, replacement, opts)
				if err := os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")+"\n"), 0644); err != nil {
					continue
				}
				a.editorGroup.ReloadFile(filePath)
			}
			a.search.Refresh()
			a.StatusNotify(fmt.Sprintf("Replaced %d matches across %d files", totalMatches, totalFiles))
		},
	})
}

func (a *App) OpenChangeDiff(dir string, status git.FileStatus) {
	fullPath := filepath.Join(dir, status.Path)
	if status.Status == "?" {
		a.editorGroup.OpenFile(fullPath)
		a.root.SetFocus(a.editorGroup)
		return
	}
	var diffText string
	var err error
	if status.Status == "R" && status.OldPath != "" {
		diffText, err = git.DiffRename(dir, status.OldPath, status.Path)
	} else {
		diffText, err = git.DiffFile(dir, status.Path)
	}
	if err != nil || diffText == "" {
		a.editorGroup.OpenFile(fullPath)
		a.root.SetFocus(a.editorGroup)
		return
	}
	parsed := diff.Parse(diffText)
	if len(parsed.Hunks) == 0 {
		a.editorGroup.OpenFile(fullPath)
		a.root.SetFocus(a.editorGroup)
		return
	}
	a.editorGroup.OpenDiff(status.Path, parsed)
	a.root.SetFocus(a.editorGroup)
}

func (a *App) OpenPRDiff(group *ui.ChangesGroup, status git.FileStatus) {
	diffText, ok := group.PRDiffs[status.Path]
	if !ok || diffText == "" {
		a.StatusWarn("No diff available for " + status.Path)
		return
	}
	parsed := diff.Parse(diffText)
	if len(parsed.Hunks) == 0 {
		a.StatusWarn("Empty diff for " + status.Path)
		return
	}
	a.editorGroup.OpenDiff(status.Path, parsed)
	a.root.SetFocus(a.editorGroup)
}

func (a *App) ShowPRGroupMenu(group *ui.ChangesGroup, sx, sy int) {
	reg := a.reg
	name := group.Name
	url := group.PRURL
	refreshID := "pr.refresh." + name
	closeID := "pr.close." + name
	reg.Register(command.Command{
		ID: refreshID, Title: "Refresh",
		Handler: func() {
			a.changes.RemovePRGroup(name)
			a.fetchAndOpenPR(url)
		},
	})
	reg.Register(command.Command{
		ID: closeID, Title: "Close",
		Handler: func() {
			a.changes.RemovePRGroup(name)
		},
	})
	items := []ui.ContextMenuItem{
		{Label: "Refresh", Command: refreshID},
		{Label: "Close", Command: closeID},
	}
	openContextMenu(a, items, sx, sy)
}

func (a *App) ShowGroupMenu(dir string, sx, sy int) {
	reg := a.reg
	items := []ui.ContextMenuItem{
		{Label: "Pull", Command: "git.pull." + dir},
		{Label: "Push", Command: "git.push." + dir},
		{Label: "Sync", Command: "git.sync." + dir},
	}
	registerDirGitCmd := func(id, title string, ops []func(string) error, verb string) {
		reg.Register(command.Command{
			ID: id, Title: title,
			Handler: func() {
				for _, op := range ops {
					if err := op(dir); err != nil {
						a.StatusError(fmt.Sprintf("%s failed: %v", verb, err))
						return
					}
				}
				a.StatusNotify(verb + " successfully")
				a.changes.Refresh()
			},
		})
	}
	registerDirGitCmd("git.pull."+dir, "Pull", []func(string) error{git.Pull}, "Pulled")
	registerDirGitCmd("git.push."+dir, "Push", []func(string) error{git.Push}, "Pushed")
	registerDirGitCmd("git.sync."+dir, "Sync", []func(string) error{git.Pull, git.Push}, "Synced")
	openContextMenu(a, items, sx, sy)
}

func (a *App) CommitChanges(dir string, message string) {
	if err := git.Commit(dir, message); err != nil {
		a.StatusError("Commit failed: " + err.Error())
	} else {
		for i := range a.changes.Groups {
			if a.changes.Groups[i].Dir == dir {
				a.changes.Groups[i].Input.Clear()
			}
		}
		a.StatusNotify("Committed: " + message)
		a.changes.Refresh()
	}
}

func (a *App) ConfirmDiscard(message string, onConfirm func()) {
	a.ShowConfirmDialog(message,
		[]string{"Cancel", "Discard"},
		[]func(){
			func() { a.DismissDialog() },
			func() {
				a.DismissDialog()
				onConfirm()
			},
		},
	)
}

func registerWidgetCallbacks(app *App) {
	reg := app.reg

	for i := range menuBarMenus {
		idx := i
		reg.Register(command.Command{
			ID:    menuBarLabels[idx],
			Title: "Open " + menuBarLabels[idx] + " Menu",
			Handler: func() {
				openMenuBarDropdown(app, idx)
			},
		})
	}

	app.menuBar.OnSelect = func(index int) {
		openMenuBarDropdown(app, index)
	}

	app.root.OnRightClick = func(mx, my int) {
		handleRightClick(app, mx, my)
	}

	app.splitPanel.OnLeftClick = func() {
		reg.Execute("sidebar.focus")
	}
	app.splitPanel.OnRightClick = func() {}

	app.sidebar.MoreButton.OnClick = app.ShowSidebarMoreMenu

	app.sidebar.OnPanelChange = func(id string) {
		if id == "search" {
			app.applySearchHighlights()
		} else {
			app.editorGroup.ClearSearch()
		}
		if id == "changes" {
			app.changes.Refresh()
		}
	}

	app.sidebar.OnTabOverflow = func(ids []string, titles []string, sx, sy int) {
		var items []ui.ContextMenuItem
		for i, id := range ids {
			panelID := id
			items = append(items, ui.ContextMenuItem{Label: titles[i], Command: "sidebar.overflow." + panelID})
			reg.Register(command.Command{
				ID:      "sidebar.overflow." + panelID,
				Title:   titles[i],
				Handler: func() { app.sidebar.SetActivePanel(panelID) },
			})
		}
		openContextMenu(app, items, sx, sy)
	}

	app.editorGroup.TabBar.OnTabClose = func(index int) {
		app.editorGroup.SwitchTab(index)
		reg.Execute("tab.close")
	}

	app.editorGroup.TabBar.MoreButton.OnClick = func(sx, sy int) {
		moreMenu := []ui.ContextMenuItem{
			{Label: "Close All", Command: "tab.closeAll"},
		}
		openContextMenu(app, moreMenu, sx, sy)
	}

	app.editorGroup.TabBar.OnTabRightClick = func(index, sx, sy int) {
		app.editorGroup.SwitchTab(index)
		tabContextMenu := []ui.ContextMenuItem{
			{Label: "Close", Shortcut: app.KeyFor("tab.close"), Command: "tab.close"},
			{Label: "Close Others", Shortcut: "", Command: "tab.closeOthers"},
			{Label: "Close All", Shortcut: "", Command: "tab.closeAll"},
		}
		openContextMenu(app, tabContextMenu, sx, sy)
	}

	app.explorer.OnOpenFile = func(path string) {
		app.editorGroup.OpenFile(path)
		app.root.SetFocus(app.editorGroup)
	}

	app.search.OnClear = func() {
		app.editorGroup.ClearSearch()
	}
	app.search.PostEvent = func() {
		app.screen.PostEvent(tcell.NewEventInterrupt(nil))
	}
	app.search.DiffSources = app.DiffSearchSources
	app.search.OnOpenMatch = app.NavigateToSearchMatch
	app.search.OnPreview = app.PreviewSearchReplace
	app.search.OnReplace = app.ApplySearchReplace
	app.search.OnReplaceAll = app.ApplySearchReplaceAll

	app.explorer.OnRightClick = func(node *ui.TreeNode, sx, sy int) {
		items := []ui.ContextMenuItem{
			{Label: "Open", Command: "explorer.open"},
			ui.MenuSep(),
			{Label: "New File", Command: "explorer.newFile"},
			{Label: "New Folder", Command: "explorer.newFolder"},
			ui.MenuSep(),
			{Label: "Rename", Command: "explorer.rename"},
			{Label: "Delete", Command: "explorer.delete"},
		}
		openContextMenu(app, items, sx, sy)
	}

	app.changes.OnRightClick = func(dir string, status git.FileStatus, sx, sy int) {
		if status.Staged {
			openContextMenu(app, changesContextMenuStaged, sx, sy)
		} else {
			openContextMenu(app, changesContextMenuUnstaged, sx, sy)
		}
	}

	app.changes.OnOpenDiff = app.OpenChangeDiff
	app.changes.OnOpenPRDiff = app.OpenPRDiff
	app.changes.OnPRGroupMenu = app.ShowPRGroupMenu
	app.changes.OnGroupMenu = app.ShowGroupMenu
	app.changes.OnCommit = app.CommitChanges
	app.changes.OnConfirmDiscard = app.ConfirmDiscard

	app.contentSplit.OnResize = func(height int) {
		if height <= 0 {
			app.contentSplit.ShowBottom = false
		} else {
			app.contentSplit.ShowBottom = true
			app.contentSplit.BottomH = height
			if len(app.terminals) == 0 {
				app.SpawnTerminal()
			}
		}
	}

	app.contentSplit.OnTopClick = func() {
		app.root.SetFocus(app.editorGroup)
	}

	app.contentSplit.OnBottomClick = func() {
		if w := app.bottomPanel.ActiveWidget(); w != nil {
			app.root.SetFocus(w)
		}
	}

	app.splitPanel.OnResize = func(width int) {
		app.SetSidebarWidth(width)
	}

	app.bottomPanel.TabBar.OnTabClick = func(index int) {
		panels := app.bottomPanel.PanelIDs()
		if index >= 0 && index < len(panels) {
			app.bottomPanel.SetActivePanel(panels[index])
			if w := app.bottomPanel.ActiveWidget(); w != nil {
				app.root.SetFocus(w)
			}
		}
	}

	app.bottomPanel.TabBar.OnAdd = func() {
		reg.Execute("terminal.new")
	}

	app.bottomPanel.TabBar.MoreButton = ui.NewMoreButtonWidget()
	app.bottomPanel.TabBar.MoreButton.OnClick = func(sx, sy int) {
		items := []ui.ContextMenuItem{
			{Label: "New Terminal", Command: "terminal.new"},
			ui.MenuSep(),
			{Label: "Close All Terminals", Command: "terminal.closeAll"},
		}
		openContextMenu(app, items, sx, sy)
	}
}
