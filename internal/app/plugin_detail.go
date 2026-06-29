package app

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eugenioenko/ttt/internal/plugin"
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"

	"github.com/gdamore/tcell/v2"
)

type pluginReadmeResult struct {
	name    string
	content string
	err     error
}

type pluginDetailState struct {
	markdown   *widgets.MarkdownWidget
	installBtn *widgets.ButtonWidget
}

func (a *App) OpenPluginDetail(entry plugin.RemoteRegistryEntry) {
	tabID := "plugin-detail:" + entry.Name

	md := widgets.NewMarkdownWidget()
	md.SetContent("Loading README...")

	installed := a.isPluginInstalled(entry.Name)

	installBtn := widgets.NewButtonWidget(widgets.ButtonConfig{
		Label: "Install",
		Style: term.StyleDefault,
		OnClick: func() {
			if a.PluginsPanel != nil && a.PluginsPanel.OnInstall != nil {
				a.PluginsPanel.OnInstall(entry.Repo, entry.Path, entry.Name)
				if state, ok := a.pluginDetailWidgets[entry.Name]; ok {
					state.installBtn.SetLabel("Installing...")
					state.installBtn.Disabled = true
				}
			}
		},
	})

	if installed {
		installBtn.SetLabel("Installed")
		installBtn.Disabled = true
	}

	a.pluginDetailWidgets[entry.Name] = &pluginDetailState{
		markdown:   md,
		installBtn: installBtn,
	}

	nameLabel := widgets.NewLabelWidget(widgets.LabelConfig{
		Text:  entry.Name,
		Style: term.StyleHoverBold,
	})

	headerRow := widgets.NewHStackWidget(nameLabel, installBtn)
	headerRow.FixedHeight = 1

	var metaParts []string
	if entry.Version != "" {
		metaParts = append(metaParts, "v"+entry.Version)
	}
	metaParts = append(metaParts, entry.Author)
	metaLabel := widgets.NewLabelWidget(widgets.LabelConfig{
		Text:  strings.Join(metaParts, " · "),
		Style: term.StyleMuted,
	})

	descLabel := widgets.NewParagraphWidget(entry.Description)
	descLabel.Style = term.StyleDefault

	headerWidgets := []widgets.Widget{headerRow, metaLabel, descLabel}

	if len(entry.Tags) > 0 {
		tagsLabel := widgets.NewLabelWidget(widgets.LabelConfig{
			Text:  strings.Join(entry.Tags, " · "),
			Style: term.StyleMuted,
		})
		headerWidgets = append(headerWidgets, tagsLabel)
	}

	header := widgets.NewVStackWidget(headerWidgets...)
	header.Box.PaddingLeft = 1
	header.Box.PaddingRight = 1

	divider := widgets.NewDividerWidget(widgets.DividerConfig{})

	scroll := widgets.NewScrollViewWidget(md)
	scroll.Box.PaddingLeft = 1
	scroll.Box.PaddingRight = 1

	content := widgets.NewVStackWidget(header, divider, scroll)
	adapter := ui.NewWidgetAdapter(content)

	a.EditorGroup.OpenPluginTab(tabID, entry.Name, adapter)

	go func() {
		readme, err := fetchPluginReadme(entry.Repo, entry.Path)
		a.Screen.PostEvent(tcell.NewEventInterrupt(&pluginReadmeResult{
			name:    entry.Name,
			content: readme,
			err:     err,
		}))
	}()
}

func (a *App) handlePluginReadmeResult(result *pluginReadmeResult) {
	state, ok := a.pluginDetailWidgets[result.name]
	if !ok {
		return
	}
	if result.err != nil {
		state.markdown.SetContent(fmt.Sprintf("Could not load README: %s", result.err))
	} else {
		state.markdown.SetContent(result.content)
	}
}

func (a *App) updatePluginDetailButtons() {
	for name, state := range a.pluginDetailWidgets {
		if a.isPluginInstalled(name) {
			state.installBtn.SetLabel("Installed")
			state.installBtn.Disabled = true
		}
	}
}

func (a *App) cleanupPluginDetailTab(id string) {
	const prefix = "plugin-detail:"
	if name, ok := strings.CutPrefix(id, prefix); ok {
		delete(a.pluginDetailWidgets, name)
	}
}

func (a *App) isPluginInstalled(name string) bool {
	for _, n := range a.PluginManager.InstalledPluginNames() {
		if n == name {
			return true
		}
	}
	return false
}

func fetchPluginReadme(repoURL, repoPath string) (string, error) {
	rawURL := repoToRawReadme(repoURL, repoPath)
	if rawURL == "" {
		return "", fmt.Errorf("unsupported repository URL")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch readme: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("readme returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read readme: %w", err)
	}

	return string(body), nil
}

func repoToRawReadme(repoURL, repoPath string) string {
	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")

	if path, ok := strings.CutPrefix(repoURL, "https://github.com/"); ok {
		prefix := "https://raw.githubusercontent.com/" + path + "/main/"
		if repoPath != "" {
			return prefix + repoPath + "/README.md"
		}
		return prefix + "README.md"
	}

	return ""
}
