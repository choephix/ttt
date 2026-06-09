package app

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
	switch a.Sidebar.ActivePanel {
	case "explorer":
		items = []ui.ContextMenuItem{
			{Label: "New File", Command: "file.new"},
			{Label: "Add Folder", Command: "workspace.addFolder"},
			{Label: "Refresh", Command: "explorer.refresh"},
		}
	case "search":
		replaceLabel := "Replace"
		if a.Search.IsReplaceMode() {
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
	sources := a.EditorGroup.DiffTabSources()
	for _, s := range sources {
		seen[s.TabName] = true
	}
	for _, g := range a.Changes.Groups {
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
		if !a.EditorGroup.SwitchToTabByPath(path) {
			filePath := strings.TrimSuffix(path, " (diff)")
			for _, g := range a.Changes.Groups {
				if !g.IsPR {
					continue
				}
				if diffText, ok := g.PRDiffs[filePath]; ok {
					a.EditorGroup.OpenDiff(filePath, diff.Parse(diffText))
					break
				}
			}
		}
		if dv := a.EditorGroup.ActiveDiffWidget(); dv != nil {
			dv.ScrollToLine(line - 1)
			dv.ApplySearchHighlight(a.Search.Input.Text, a.Search.Options)
		}
		a.Root.SetFocus(a.EditorGroup)
		return
	}
	a.EditorGroup.OpenFile(path)
	a.EditorGroup.GoToLine(line)
	if a.Search.Input.Text != "" {
		matches, _ := ui.FindInLines(a.EditorGroup.Editor.Buf.Lines, a.Search.Input.Text, a.Search.Options)
		a.EditorGroup.SetSearch(a.Search.Input.Text, matches)
	}
	a.Root.SetFocus(a.EditorGroup)
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
	a.EditorGroup.OpenDiff(filePath, fd)
	a.Root.SetFocus(a.EditorGroup)
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
	a.EditorGroup.ReloadFile(filePath)
	a.Search.Refresh()
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
				a.EditorGroup.ReloadFile(filePath)
			}
			a.Search.Refresh()
			a.StatusNotify(fmt.Sprintf("Replaced %d matches across %d files", totalMatches, totalFiles))
		},
	})
}

func (a *App) OpenChangeDiff(dir string, status git.FileStatus) {
	fullPath := filepath.Join(dir, status.Path)
	if status.Status == "?" {
		a.EditorGroup.OpenFile(fullPath)
		a.Root.SetFocus(a.EditorGroup)
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
		a.EditorGroup.OpenFile(fullPath)
		a.Root.SetFocus(a.EditorGroup)
		return
	}
	parsed := diff.Parse(diffText)
	if len(parsed.Hunks) == 0 {
		a.EditorGroup.OpenFile(fullPath)
		a.Root.SetFocus(a.EditorGroup)
		return
	}
	a.EditorGroup.OpenDiff(status.Path, parsed)
	a.Root.SetFocus(a.EditorGroup)
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
	a.EditorGroup.OpenDiff(status.Path, parsed)
	a.Root.SetFocus(a.EditorGroup)
}

func (a *App) ShowPRGroupMenu(group *ui.ChangesGroup, sx, sy int) {
	reg := a.Reg
	name := group.Name
	url := group.PRURL
	refreshID := "pr.refresh." + name
	closeID := "pr.close." + name
	reg.Register(command.Command{
		ID: refreshID, Title: "Refresh",
		Handler: func() {
			a.Changes.RemovePRGroup(name)
			a.FetchAndOpenPR(url)
		},
	})
	reg.Register(command.Command{
		ID: closeID, Title: "Close",
		Handler: func() {
			a.Changes.RemovePRGroup(name)
		},
	})
	items := []ui.ContextMenuItem{
		{Label: "Refresh", Command: refreshID},
		{Label: "Close", Command: closeID},
	}
	openContextMenu(a, items, sx, sy)
}

func (a *App) ShowGroupMenu(dir string, sx, sy int) {
	reg := a.Reg
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
				a.Changes.Refresh()
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
		for i := range a.Changes.Groups {
			if a.Changes.Groups[i].Dir == dir {
				a.Changes.Groups[i].Input.Clear()
			}
		}
		a.StatusNotify("Committed: " + message)
		a.Changes.Refresh()
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
	reg := app.Reg

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

	app.MenuBar.OnSelect = func(index int) {
		openMenuBarDropdown(app, index)
	}

	app.Root.OnRightClick = func(mx, my int) {
		handleRightClick(app, mx, my)
	}

	app.SplitPanel.OnLeftClick = func() {
		reg.Execute("sidebar.focus")
	}
	app.SplitPanel.OnRightClick = func() {}

	app.Sidebar.MoreButton.OnClick = app.ShowSidebarMoreMenu

	app.Sidebar.OnPanelChange = func(id string) {
		if id == "search" {
			app.applySearchHighlights()
		} else {
			app.EditorGroup.ClearSearch()
		}
		if id == "changes" {
			app.Changes.Refresh()
		}
	}

	app.Sidebar.OnTabOverflow = func(ids []string, titles []string, sx, sy int) {
		var items []ui.ContextMenuItem
		for i, id := range ids {
			panelID := id
			items = append(items, ui.ContextMenuItem{Label: titles[i], Command: "sidebar.overflow." + panelID})
			reg.Register(command.Command{
				ID:      "sidebar.overflow." + panelID,
				Title:   titles[i],
				Handler: func() { app.Sidebar.SetActivePanel(panelID) },
			})
		}
		openContextMenu(app, items, sx, sy)
	}

	app.EditorGroup.TabBar.OnTabClose = func(index int) {
		app.EditorGroup.SwitchTab(index)
		reg.Execute("tab.close")
	}

	app.EditorGroup.TabBar.MoreButton.OnClick = func(sx, sy int) {
		moreMenu := []ui.ContextMenuItem{
			{Label: "Close All", Command: "tab.closeAll"},
		}
		openContextMenu(app, moreMenu, sx, sy)
	}

	app.EditorGroup.TabBar.OnTabRightClick = func(index, sx, sy int) {
		app.EditorGroup.SwitchTab(index)
		tabContextMenu := []ui.ContextMenuItem{
			{Label: "Close", Shortcut: app.KeyFor("tab.close"), Command: "tab.close"},
			{Label: "Close Others", Shortcut: "", Command: "tab.closeOthers"},
			{Label: "Close All", Shortcut: "", Command: "tab.closeAll"},
		}
		openContextMenu(app, tabContextMenu, sx, sy)
	}

	app.Explorer.OnOpenFile = func(path string) {
		app.EditorGroup.OpenFile(path)
		app.Root.SetFocus(app.EditorGroup)
	}

	app.Search.OnClear = func() {
		app.EditorGroup.ClearSearch()
	}
	app.Search.PostBatch = func(batch *ui.SearchBatch) {
		app.Screen.PostEvent(tcell.NewEventInterrupt(batch))
	}
	app.Search.DiffSources = app.DiffSearchSources
	app.Search.OnOpenMatch = app.NavigateToSearchMatch
	app.Search.OnPreview = app.PreviewSearchReplace
	app.Search.OnReplace = app.ApplySearchReplace
	app.Search.OnReplaceAll = app.ApplySearchReplaceAll

	app.Explorer.OnRightClick = func(node *ui.TreeNode, sx, sy int) {
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

	app.Changes.OnRightClick = func(dir string, status git.FileStatus, sx, sy int) {
		if status.Staged {
			openContextMenu(app, changesContextMenuStaged, sx, sy)
		} else {
			openContextMenu(app, changesContextMenuUnstaged, sx, sy)
		}
	}

	app.Changes.OnOpenDiff = app.OpenChangeDiff
	app.Changes.OnOpenPRDiff = app.OpenPRDiff
	app.Changes.OnPRGroupMenu = app.ShowPRGroupMenu
	app.Changes.OnGroupMenu = app.ShowGroupMenu
	app.Changes.OnCommit = app.CommitChanges
	app.Changes.OnConfirmDiscard = app.ConfirmDiscard

	app.ContentSplit.OnResize = func(height int) {
		if height <= 0 {
			app.ContentSplit.ShowBottom = false
		} else {
			app.ContentSplit.ShowBottom = true
			app.ContentSplit.BottomH = height
			if len(app.Terminals) == 0 {
				app.SpawnTerminal()
			}
		}
	}

	app.ContentSplit.OnTopClick = func() {
		app.Root.SetFocus(app.EditorGroup)
	}

	app.ContentSplit.OnBottomClick = func() {
		if w := app.BottomPanel.ActiveWidget(); w != nil {
			app.Root.SetFocus(w)
		}
	}

	app.SplitPanel.OnResize = func(width int) {
		app.SetSidebarWidth(width)
	}

	app.BottomPanel.TabBar.OnTabClick = func(index int) {
		panels := app.BottomPanel.PanelIDs()
		if index >= 0 && index < len(panels) {
			app.BottomPanel.SetActivePanel(panels[index])
			if w := app.BottomPanel.ActiveWidget(); w != nil {
				app.Root.SetFocus(w)
			}
		}
	}

	app.BottomPanel.TabBar.OnAdd = func() {
		reg.Execute("terminal.new")
	}

	app.BottomPanel.TabBar.MoreButton = ui.NewMoreButtonWidget()
	app.BottomPanel.TabBar.MoreButton.OnClick = func(sx, sy int) {
		items := []ui.ContextMenuItem{
			{Label: "New Terminal", Command: "terminal.new"},
			ui.MenuSep(),
			{Label: "Close All Terminals", Command: "terminal.closeAll"},
		}
		openContextMenu(app, items, sx, sy)
	}
}
