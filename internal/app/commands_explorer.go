package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/core/clipboard"
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
		[]string{"No", "Yes"},
		[]func(){
			func() {
				a.DismissDialog()
			},
			func() {
				a.DismissDialog()
				if err := os.RemoveAll(node.Path); err != nil {
					a.StatusError("Error: " + err.Error())
					return
				}
				a.Explorer.Reload()
			},
		},
	)
}

func (a *App) CopyAbsolutePath() {
	path := a.activeFilePath()
	if path == "" {
		a.StatusWarn("No file open")
		return
	}
	clipboard.Set(path)
	a.StatusNotify("Absolute path copied to clipboard")
}

func (a *App) CopyRelativePath() {
	path := a.activeFilePath()
	if path == "" {
		a.StatusWarn("No file open")
		return
	}
	rel := a.relativePath(path)
	clipboard.Set(rel)
	a.StatusNotify("Relative path copied to clipboard")
}

func (a *App) ExplorerCopyAbsolutePath() {
	path := a.explorerNodePath()
	if path == "" {
		a.StatusWarn("No file selected")
		return
	}
	clipboard.Set(path)
	a.ExplorerContextNode = nil
	a.StatusNotify("Absolute path copied to clipboard")
}

func (a *App) ExplorerCopyRelativePath() {
	path := a.explorerNodePath()
	if path == "" {
		a.StatusWarn("No file selected")
		return
	}
	rel := a.relativePath(path)
	clipboard.Set(rel)
	a.ExplorerContextNode = nil
	a.StatusNotify("Relative path copied to clipboard")
}

func (a *App) activeFilePath() string {
	path := a.EditorGroup.ActiveFilePath()
	if path == "untitled" {
		return ""
	}
	path = strings.TrimSuffix(path, " (diff)")
	if !filepath.IsAbs(path) {
		if len(a.Workspace.Folders) > 0 {
			path = filepath.Join(a.Workspace.Folders[0].Path, path)
		}
	}
	return path
}

func (a *App) explorerNodePath() string {
	if a.ExplorerContextNode != nil {
		return a.ExplorerContextNode.Path
	}
	if node := a.Explorer.SelectedNode(); node != nil {
		return node.Path
	}
	return ""
}

func (a *App) relativePath(absPath string) string {
	if folder := a.Workspace.FolderForFile(absPath); folder != nil {
		if rel, err := filepath.Rel(folder.Path, absPath); err == nil {
			return rel
		}
	}
	primary := a.Workspace.Primary()
	if primary != "" {
		if rel, err := filepath.Rel(primary, absPath); err == nil {
			return rel
		}
	}
	return absPath
}

func registerExplorerCommands(app *App) {
	reg := app.Reg

	reg.Register(command.Command{
		ID: "explorer.refresh", Title: "Explorer: Refresh",
		Keywords: []string{"view", "file", "reload"},
		Handler:  func() { app.Explorer.Reload() },
	})

	reg.Register(command.Command{
		ID: "explorer.open", Title: "Explorer: Toggle Node",
		Keywords: []string{"view", "file"},
		Handler:  func() { app.Explorer.ActivateSelected() },
	})

	reg.Register(command.Command{
		ID: "explorer.newFile", Title: "Explorer: New File",
		Keywords: []string{"view", "file", "create"},
		Handler:  app.ExplorerNewFile,
	})

	reg.Register(command.Command{
		ID: "explorer.newFolder", Title: "Explorer: New Folder",
		Keywords: []string{"view", "file", "create", "directory"},
		Handler:  app.ExplorerNewFolder,
	})

	reg.Register(command.Command{
		ID: "explorer.rename", Title: "Explorer: Rename",
		Keywords: []string{"view", "file"},
		Handler:  app.ExplorerRename,
	})

	reg.Register(command.Command{
		ID: "explorer.delete", Title: "Explorer: Delete",
		Keywords: []string{"view", "file", "remove"},
		Handler:  app.ExplorerDelete,
	})

	reg.Register(command.Command{
		ID: "file.copyAbsolutePath", Title: "File: Copy Absolute Path",
		Keywords: []string{"file", "path", "copy", "clipboard", "absolute"},
		Handler:  app.CopyAbsolutePath,
	})

	reg.Register(command.Command{
		ID: "file.copyRelativePath", Title: "File: Copy Relative Path",
		Keywords: []string{"file", "path", "copy", "clipboard", "relative"},
		Handler:  app.CopyRelativePath,
	})

	reg.Register(command.Command{
		ID: "explorer.copyAbsolutePath", Title: "Explorer: Copy Absolute Path",
		Keywords: []string{"explorer", "file", "path", "copy", "clipboard", "absolute"},
		Handler:  app.ExplorerCopyAbsolutePath,
	})

	reg.Register(command.Command{
		ID: "explorer.copyRelativePath", Title: "Explorer: Copy Relative Path",
		Keywords: []string{"explorer", "file", "path", "copy", "clipboard", "relative"},
		Handler:  app.ExplorerCopyRelativePath,
	})
}
