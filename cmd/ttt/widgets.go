package main

import (
	"os"
	"path/filepath"
	"ttt/internal/config"
	"ttt/internal/git"
	"ttt/internal/term"
	"ttt/internal/ui"
	"ttt/internal/view"
)

func resolveArgs() (workDir string, openFile string) {
	workDir, _ = os.Getwd()
	if len(os.Args) < 2 {
		return
	}
	arg := os.Args[1]
	absPath, err := filepath.Abs(arg)
	if err != nil {
		openFile = arg
		return
	}
	info, err := os.Stat(absPath)
	if err != nil {
		openFile = absPath
		return
	}
	if info.IsDir() {
		workDir = absPath
	} else {
		openFile = absPath
		if root := git.RepoRoot(filepath.Dir(absPath)); root != "" {
			workDir = root
		} else {
			workDir = filepath.Dir(absPath)
		}
	}
	return
}

func buildWidgets(cfg *config.AppConfig, borders *term.BorderSet) *appWidgets {
	workDir, openFile := resolveArgs()

	editorGroup := ui.NewEditorGroupWidget(borders, cfg.Settings.TabSize, cfg.Settings.LineNumbers)
	if openFile != "" {
		editorGroup.OpenFile(openFile)
	}

	bottomPanel := ui.NewBottomPanelWidget(borders)
	bottomPanel.AddPanel("output", "OUTPUT", ui.NewPlaceholderWidget(""))
	bottomPanel.AddPanel("debug", "DEBUG", ui.NewPlaceholderWidget(""))
	bottomPanel.AddPanel("terminal", "TERMINAL", ui.NewPlaceholderWidget(""))
	bottomPanel.AddPanel("ports", "PORTS", ui.NewPlaceholderWidget(""))

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

	explorer := ui.NewExplorerWidget(workDir)
	search := ui.NewSearchWidget()
	changes := ui.NewChangesWidget(workDir)

	sidebar := ui.NewSidebarWidget()
	sidebar.AddPanel("explorer", "Files", explorer)
	sidebar.AddPanel("search", "Search", search)
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

	rootBox := &ui.VBox{}
	rootBox.AddChild(menuBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})
	rootBox.AddChild(splitPanel, ui.LayoutConstraint{Type: ui.Flex, Value: 1})
	rootBox.AddChild(statusBar, ui.LayoutConstraint{Type: ui.Fixed, Value: 1})

	root := ui.NewRoot(rootBox)
	root.SetFocus(editorGroup)

	app := &appWidgets{
		root:         root,
		editorGroup:  editorGroup,
		sidebar:      sidebar,
		splitPanel:   splitPanel,
		contentSplit: contentSplit,
		bottomPanel:  bottomPanel,
		explorer:     explorer,
		search:       search,
		changes:      changes,
		statusBar:    statusBar,
		status:       status,
		borders:      borders,
		cwd:          workDir,
	}
	app.showSidebar = func() {
		sidebar.Visible = true
		splitPanel.ShowLeft = true
	}
	app.hideSidebar = func() {
		sidebar.Visible = false
		splitPanel.ShowLeft = false
	}
	app.setSidebarWidth = func(w int) {
		if w <= 0 {
			app.hideSidebar()
			return
		}
		if !sidebar.Visible {
			app.showSidebar()
		}
		sidebarWidth = w
		splitPanel.DividerPos = sidebarWidth
	}

	return app
}
