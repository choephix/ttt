package app

import (
	"os"
	"path/filepath"

	"github.com/eugenioenko/ttt/internal/command"
)

func (a *App) ExplorerNewFile() {
	node := a.Explorer.SelectedNode()
	if node == nil {
		return
	}
	parentDir := node.Path
	if !node.IsDir {
		parentDir = filepath.Dir(node.Path)
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
		a.Explorer.Reload()
		a.EditorGroup.OpenFile(newPath)
		a.FocusEditor()
	})
}

func (a *App) ExplorerNewFolder() {
	node := a.Explorer.SelectedNode()
	if node == nil {
		return
	}
	parentDir := node.Path
	if !node.IsDir {
		parentDir = filepath.Dir(node.Path)
	}
	a.ShowInputDialog("New Folder", "Folder name", "", func(name string) {
		newPath := filepath.Join(parentDir, name)
		if err := os.MkdirAll(newPath, 0755); err != nil {
			a.StatusError("Error: " + err.Error())
			return
		}
		a.Explorer.Reload()
	})
}

func (a *App) ExplorerRename() {
	node := a.Explorer.SelectedNode()
	if node == nil {
		return
	}
	a.ShowInputDialog("Rename", "New name", node.Name, func(newName string) {
		dir := filepath.Dir(node.Path)
		newPath := filepath.Join(dir, newName)
		if err := os.Rename(node.Path, newPath); err != nil {
			a.StatusError("Error: " + err.Error())
			return
		}
		a.Explorer.Reload()
	})
}

func (a *App) ExplorerDelete() {
	node := a.Explorer.SelectedNode()
	if node == nil {
		return
	}
	a.ShowConfirmDialog("Delete "+node.Name+"?",
		[]string{"Yes", "No"},
		[]func(){
			func() {
				a.DismissDialog()
				if err := os.RemoveAll(node.Path); err != nil {
					a.StatusError("Error: " + err.Error())
					return
				}
				a.Explorer.Reload()
			},
			func() { a.DismissDialog() },
		},
	)
}

func registerExplorerCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "explorer.refresh", Title: "Refresh Explorer",
		Handler: func() { app.Explorer.Reload() },
	})

	reg.Register(command.Command{
		ID: "explorer.open", Title: "Open",
		Handler: func() { app.Explorer.ActivateSelected() },
	})

	reg.Register(command.Command{
		ID: "explorer.newFile", Title: "New File",
		Handler: app.ExplorerNewFile,
	})

	reg.Register(command.Command{
		ID: "explorer.newFolder", Title: "New Folder",
		Handler: app.ExplorerNewFolder,
	})

	reg.Register(command.Command{
		ID: "explorer.rename", Title: "Rename",
		Handler: app.ExplorerRename,
	})

	reg.Register(command.Command{
		ID: "explorer.delete", Title: "Delete",
		Handler: app.ExplorerDelete,
	})
}
