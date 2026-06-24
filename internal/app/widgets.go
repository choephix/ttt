package app

import (
	"os"
	"path/filepath"
	"strings"
	"github.com/eugenioenko/ttt/internal/config"
	"github.com/eugenioenko/ttt/internal/github"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/view"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/eugenioenko/ttt/internal/workspace"
)

func isPRURL(arg string) bool {
	return strings.Contains(arg, "github.com/") && strings.Contains(arg, "/pull/")
}

func resolveArgs() (ws *workspace.Workspace, openFiles []string, configFile string, prURLs []string) {
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
		if isPRURL(args[i]) {
			if _, _, _, err := github.ParsePRURL(args[i]); err == nil {
				prURLs = append(prURLs, args[i])
			}
			continue
		}
		absPath, err := filepath.Abs(workspace.ExpandPath(args[i]))
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

	if len(folders) == 0 && len(prURLs) == 0 && len(openFiles) == 0 {
		cwd, _ := os.Getwd()
		folders = append(folders, cwd)
	}
	ws = workspace.New(folders)
	return
}

func BuildApp(cfg *config.AppConfig, borders *term.BorderSet) (*App, []string) {
	ws, openFiles, _, prURLs := resolveArgs()
	return BuildAppFromConfig(cfg, borders, ws, openFiles), prURLs
}

var bracketColorSlots = []term.Style{
	term.StyleBracketColor1,
	term.StyleBracketColor2,
	term.StyleBracketColor3,
	term.StyleBracketColor4,
	term.StyleBracketColor5,
	term.StyleBracketColor6,
}

func ResolveBracketColorStyles(colors []string) []term.Style {
	if len(colors) == 0 {
		colors = []string{"yellow", "magenta", "blue"}
	}
	n := len(colors)
	if n > len(bracketColorSlots) {
		n = len(bracketColorSlots)
	}
	return bracketColorSlots[:n]
}

func BuildAppFromConfig(cfg *config.AppConfig, borders *term.BorderSet, ws *workspace.Workspace, openFiles []string) *App {

	bracketStyles := ResolveBracketColorStyles(cfg.Theme.Editor.BracketColors)

	editorGroup := ui.NewEditorGroupWidget(borders, cfg.Settings.Editor.TabSize, cfg.Settings.Editor.LineNumbers, cfg.Settings.Editor.GutterStyle)
	editorGroup.InsertSpaces = cfg.Settings.Editor.InsertSpaces
	editorGroup.InsertFinalNewline = cfg.Settings.Editor.InsertFinalNewline
	editorGroup.TrimTrailingWhitespace = cfg.Settings.Editor.TrimTrailingWhitespace
	editorGroup.SyntaxHighlight = cfg.Settings.Editor.IsSyntaxHighlightEnabled()
	editorGroup.WordWrap = cfg.Settings.Editor.WordWrap
	editorGroup.Editor.WordWrap = cfg.Settings.Editor.WordWrap
	editorGroup.BracketPairColorization = cfg.Settings.Editor.BracketPairColorization
	editorGroup.Editor.BracketPairColorization = cfg.Settings.Editor.BracketPairColorization
	editorGroup.BracketColorStyles = bracketStyles
	editorGroup.Editor.BracketColorStyles = bracketStyles
	for _, f := range openFiles {
		editorGroup.OpenFile(f)
		editorGroup.PinActiveTab()
	}

	terminalPanel := ui.NewTerminalPanelWidget()
	terminalPanel.Borders = borders
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
		{Name: "Options"},
		{Name: "Help"},
	})

	explorer := ui.NewExplorerWidget(cfg.Settings.Explorer, ws.Paths()...)
	search := ui.NewSearchWidget()
	search.SetWorkDirs(ws.Paths())
	search.Debounce.DelayMs = cfg.Settings.Search.Debounce
	changes := ui.NewChangesWidget(ws.Paths()...)

	demoTree := widgets.NewTreeWidget(widgets.TreeConfig{
		Items: []*widgets.TreeNode{
			{
				ID:       "containers",
				Label:    "Containers (8)",
				Expanded: true,
				Children: []*widgets.TreeNode{
					{ID: "c1", Label: "nginx-proxy", Icon: "●", Actions: []widgets.Action{{Icon: "▶", Command: "docker.restart"}, {Icon: "■", Command: "docker.stop"}}},
					{ID: "c2", Label: "postgres-db", Icon: "●", Actions: []widgets.Action{{Icon: "▶", Command: "docker.restart"}, {Icon: "■", Command: "docker.stop"}}},
					{ID: "c3", Label: "redis-cache", Icon: "○", Actions: []widgets.Action{{Icon: "▶", Command: "docker.start"}}},
					{ID: "c4", Label: "grafana", Icon: "●", Actions: []widgets.Action{{Icon: "▶", Command: "docker.restart"}, {Icon: "■", Command: "docker.stop"}}},
					{ID: "c5", Label: "prometheus", Icon: "●", Actions: []widgets.Action{{Icon: "▶", Command: "docker.restart"}, {Icon: "■", Command: "docker.stop"}}},
					{ID: "c6", Label: "rabbitmq", Icon: "○", Actions: []widgets.Action{{Icon: "▶", Command: "docker.start"}}},
					{ID: "c7", Label: "elasticsearch", Icon: "●", Actions: []widgets.Action{{Icon: "▶", Command: "docker.restart"}, {Icon: "■", Command: "docker.stop"}}},
					{ID: "c8", Label: "kibana", Icon: "○", Actions: []widgets.Action{{Icon: "▶", Command: "docker.start"}}},
				},
			},
			{
				ID:       "images",
				Label:    "Images (10)",
				Expanded: true,
				Children: []*widgets.TreeNode{
					{ID: "i1", Label: "nginx:latest", Badge: "45MB"},
					{ID: "i2", Label: "postgres:16", Badge: "120MB"},
					{ID: "i3", Label: "redis:7", Badge: "30MB"},
					{ID: "i4", Label: "node:20", Badge: "350MB"},
					{ID: "i5", Label: "alpine:3.19", Badge: "7MB"},
					{ID: "i6", Label: "grafana:latest", Badge: "280MB"},
					{ID: "i7", Label: "prom/prometheus", Badge: "190MB"},
					{ID: "i8", Label: "rabbitmq:3", Badge: "150MB"},
					{ID: "i9", Label: "elasticsearch:8", Badge: "800MB"},
					{ID: "i10", Label: "kibana:8", Badge: "700MB"},
				},
			},
			{
				ID:       "volumes",
				Label:    "Volumes (5)",
				Expanded: true,
				Children: []*widgets.TreeNode{
					{ID: "v1", Label: "pg_data"},
					{ID: "v2", Label: "redis_data"},
					{ID: "v3", Label: "es_data"},
					{ID: "v4", Label: "grafana_data"},
					{ID: "v5", Label: "rabbitmq_data"},
				},
			},
			{
				ID:       "networks",
				Label:    "Networks (3)",
				Expanded: true,
				Children: []*widgets.TreeNode{
					{
						ID:       "n1",
						Label:    "frontend",
						Expanded: true,
						Children: []*widgets.TreeNode{
							{ID: "n1a", Label: "nginx-proxy"},
							{ID: "n1b", Label: "grafana"},
							{ID: "n1c", Label: "kibana"},
						},
					},
					{
						ID:       "n2",
						Label:    "backend",
						Expanded: true,
						Children: []*widgets.TreeNode{
							{ID: "n2a", Label: "postgres-db"},
							{ID: "n2b", Label: "redis-cache"},
							{ID: "n2c", Label: "rabbitmq"},
							{ID: "n2d", Label: "elasticsearch"},
						},
					},
					{
						ID:       "n3",
						Label:    "monitoring",
						Expanded: true,
						Children: []*widgets.TreeNode{
							{ID: "n3a", Label: "prometheus"},
							{ID: "n3b", Label: "grafana"},
						},
					},
				},
			},
		},
		NodeMenu: []widgets.MenuEntry{
			{Label: "Stop", Command: "docker.stop"},
			{Label: "Restart", Command: "docker.restart"},
			{Separator: true},
			{Label: "Remove", Command: "docker.remove"},
		},
	})
	widgetPanel := ui.NewTreeWidgetAdapter(demoTree)

	sidebar := ui.NewSidebarWidget()
	sidebar.AddPanel("explorer", "Explore", explorer)
	sidebar.AddPanel("search", "Find", search)
	sidebar.AddPanel("changes", "Changes", changes)
	sidebar.AddPanel("widgets", "Widget", widgetPanel)
	hasFolders := len(ws.Paths()) > 0
	sidebar.Visible = hasFolders
	sidebar.Borders = borders

	splitPanel := ui.NewSplitPanelWidget()
	splitPanel.Left = sidebar
	splitPanel.Right = contentSplit
	splitPanel.Borders = borders
	splitPanel.DividerPos = ui.DefaultSidebarWidth
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
		Root:              root,
		EditorGroup:       editorGroup,
		Sidebar:           sidebar,
		SplitPanel:        splitPanel,
		ContentSplit:      contentSplit,
		BottomPanel:       bottomPanel,
		Explorer:          explorer,
		Search:            search,
		Changes:           changes,
		MenuBar:           menuBar,
		StatusBar:         statusBar,
		Status:            status,
		Borders:           borders,
		Settings:          &cfg.Settings,
		Workspace:         ws,
		Palette:           BuildTerminalPalettePtr(cfg.Theme),
		TerminalPanel:     terminalPanel,
		Problems:          problems,
		WidgetPanel:       widgetPanel,
		References:        references,
		DocVersions:       make(map[string]int),
		AllDiagnostics:    make(map[string][]ui.Diagnostic),
		LspNotified:       make(map[string]bool),
	}
}
