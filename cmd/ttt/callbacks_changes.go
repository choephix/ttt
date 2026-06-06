package main

import (
	"fmt"
	"path/filepath"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/core/diff"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/ui"
)

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
