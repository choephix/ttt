package app

import (
	"os"
	"path/filepath"

	"github.com/eugenioenko/ttt/internal/core/clipboard"
)

func (a *App) FileOpNewFile(path string, reload func()) {
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	parentDir := path
	if err != nil || !info.IsDir() {
		parentDir = filepath.Dir(path)
	}
	a.ShowInputDialog("New File", "Filename", "", func(name string) {
		newPath := filepath.Join(parentDir, name)
		if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
			a.StatusError("Error: " + err.Error())
			return
		}
		if err := os.WriteFile(newPath, []byte{}, 0644); err != nil {
			a.StatusError("Error: " + err.Error())
			return
		}
		reload()
		a.EditorGroup.OpenFile(newPath)
		a.FocusEditor()
	})
}

func (a *App) FileOpNewFolder(path string, reload func()) {
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	parentDir := path
	if err != nil || !info.IsDir() {
		parentDir = filepath.Dir(path)
	}
	a.ShowInputDialog("New Folder", "Folder name", "", func(name string) {
		newPath := filepath.Join(parentDir, name)
		if err := os.MkdirAll(newPath, 0755); err != nil {
			a.StatusError("Error: " + err.Error())
			return
		}
		reload()
	})
}

func (a *App) FileOpRename(path string, reload func()) {
	if path == "" {
		return
	}
	a.ShowInputDialog("Rename", "New name", filepath.Base(path), func(newName string) {
		newPath := filepath.Join(filepath.Dir(path), newName)
		if err := os.Rename(path, newPath); err != nil {
			a.StatusError("Error: " + err.Error())
			return
		}
		reload()
	})
}

func (a *App) FileOpDelete(path string, reload func()) {
	if path == "" {
		return
	}
	name := filepath.Base(path)
	a.ShowConfirmDialog("Delete "+name+"?",
		[]string{"No", "Yes"},
		[]func(){
			func() { a.DismissDialog() },
			func() {
				a.DismissDialog()
				if err := os.RemoveAll(path); err != nil {
					a.StatusError("Error: " + err.Error())
					return
				}
				reload()
			},
		},
	)
}

func (a *App) FileOpCopyAbsolutePath(path string) {
	if path == "" {
		a.StatusWarn("No file selected")
		return
	}
	clipboard.Set(path)
	a.StatusNotify("Absolute path copied to clipboard")
}

func (a *App) FileOpCopyRelativePath(path string) {
	if path == "" {
		a.StatusWarn("No file selected")
		return
	}
	rel := a.relativePath(path)
	clipboard.Set(rel)
	a.StatusNotify("Relative path copied to clipboard")
}

func (a *App) FileOpRemoveRoot(path string) {
	if path == "" {
		return
	}
	if len(a.Workspace.Paths()) <= 1 {
		a.StatusWarn("Cannot remove the last folder")
		return
	}
	a.Workspace.RemoveFolder(path)
	a.refreshWorkspaceWidgets()
}
