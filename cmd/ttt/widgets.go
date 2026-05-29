package main

import (
	"os"
	"path/filepath"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/view"
	"github.com/eugenioenko/ttt/internal/workspace"
)

func resolveArgs() (ws *workspace.Workspace, openFiles []string, configFile string) {
	var folders []string
	var wsFile string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--workspace" && i+1 < len(args) {
			wsFile = args[i+1]
			i++
			continue
		}
		if args[i] == "--config" && i+1 < len(args) {
			configFile = args[i+1]
			i++
			continue
		}
		absPath, err := filepath.Abs(args[i])
		if err != nil {
			openFiles = append(openFiles, args[i])
			continue
		}
		info, err := os.Stat(absPath)
		if err != nil {
			openFiles = append(openFiles, absPath)
			continue
		}
		if info.IsDir() {
			folders = append(folders, absPath)
		} else {
			openFiles = append(openFiles, absPath)
			dir := filepath.Dir(absPath)
			if root := git.RepoRoot(dir); root != "" {
				folders = append(folders, root)
			} else {
				folders = append(folders, dir)
			}
		}
	}

	if wsFile != "" {
		loaded, err := workspace.LoadFile(wsFile)
		if err == nil {
			ws = loaded
			for _, f := range folders {
				ws.AddFolder(f)
			}
			return
		}
	}

	if len(folders) == 0 {
		cwd, _ := os.Getwd()
		folders = append(folders, cwd)
	}
	ws = workspace.New(folders)
	return
}

func buildApp(cfg *config.AppConfig, borders *term.BorderSet) *App {
	ws, openFiles, _ := resolveArgs()
	return buildAppFromConfig(cfg, borders, ws, openFiles)
}

func buildAppFromConfig(cfg *config.AppConfig, borders *term.BorderSet, ws *workspace.Workspace, openFiles []string) *App {

	editorGroup := ui.NewEditorGroupWidget(borders, cfg.Settings.TabSize, cfg.Settings.LineNumbers)
	editorGroup.InsertFinalNewline = cfg.Settings.InsertFinalNewline
	for _, f := range openFiles {
		editorGroup.OpenFile(f)
	}

	terminalPanel := ui.NewTerminalPanelWidget()
	problems := ui.NewProblemsWidget()
	references := ui.NewReferencesWidget()
	bottomPanel := ui.NewBottomPanelWidget(borders)
	bottomPanel.AddPanel("terminal", "TERMINAL", terminalPanel)
	bottomPanel.AddPanel("problems", "PROBLEMS", problems)
	bottomPanel.AddPanel("references", "REFERENCES", references)

	contentSplit := ui.NewContentSplitWidget()
	contentSplit.Top = editorGroup
	contentSplit.Bottom = bottomPanel
	contentSplit.Borders = borders
	contentSplit.ShowBottom = false

	status := &view.StatusBar{FileName: editorGroup.ActiveFilePath()}
	statusBar := ui.NewStatusBarWidget(status)

	menuBar := ui.NewMenuBarWidget([]ui.MenuItem{
		{Name: "File"},
		{Name: "Edit"},
		{Name: "Selection"},
		{Name: "View"},
		{Name: "Help"},
	})

	explorer := ui.NewExplorerWidget(cfg.Settings.Explorer, ws.Paths()...)
	search := ui.NewSearchWidget()
	search.SetWorkDirs(ws.Paths())
	changes := ui.NewChangesWidget(ws.Paths()...)

	sidebar := ui.NewSidebarWidget()
	sidebar.AddPanel("explorer", "Explore", explorer)
	sidebar.AddPanel("search", "Find", search)
	sidebar.AddPanel("changes", "Changes", changes)
	sidebar.Visible = cfg.Settings.SidebarVisible
	sidebar.Borders = borders

	sidebarWidth := cfg.Settings.SidebarWidth
	if sidebarWidth <= 0 {
		sidebarWidth = 30
	}

	splitPanel := ui.NewSplitPanelWidget()
	splitPanel.Left = sidebar
	splitPanel.Right = contentSplit
	splitPanel.Borders = borders
	splitPanel.DividerPos = sidebarWidth
	splitPanel.ShowLeft = sidebar.Visible
	splitPanel.RightBorderStartY = 2
	contentSplit.RightBorderStartY = &splitPanel.RightBorderStartY

	rootBox := &ui.VBox{}
	rootBox.AddChild(menuBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	rootBox.AddChild(splitPanel, ui.LayoutConstraint{Type: ui.Flex, Value: 1})
	rootBox.AddChild(statusBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})

	root := ui.NewRoot(rootBox)
	root.SetFocus(editorGroup)

	return &App{
		root:         root,
		editorGroup:  editorGroup,
		sidebar:      sidebar,
		splitPanel:   splitPanel,
		contentSplit: contentSplit,
		bottomPanel:  bottomPanel,
		explorer:     explorer,
		search:       search,
		changes:      changes,
		menuBar:      menuBar,
		statusBar:    statusBar,
		status:       status,
		borders:      borders,
		settings:     &cfg.Settings,
		workspace:    ws,
		palette:      buildTerminalPalettePtr(cfg.Theme),
		terminalPanel:  terminalPanel,
		problems:       problems,
		references:     references,
		docVersions:    make(map[string]int),
		allDiagnostics: make(map[string][]ui.Diagnostic),
		lspNotified:    make(map[string]bool),
	}
}
