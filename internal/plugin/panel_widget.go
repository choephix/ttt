package plugin

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type PluginPanelWidget struct {
	widgets.BaseWidget
	plugin *Plugin
}

func NewPluginPanelWidget(p *Plugin) *PluginPanelWidget {
	return &PluginPanelWidget{plugin: p}
}

func (pw *PluginPanelWidget) Height() int { return 0 }
func (pw *PluginPanelWidget) Width() int  { return 0 }

func (pw *PluginPanelWidget) Render(surface widgets.Surface) {
	w, h := surface.Size()
	if w <= 0 || h <= 0 {
		return
	}

	surface.Fill(term.Cell{Ch: ' ', Style: term.StyleDefault})

	proxy := NewPanelProxy(surface)
	if err := pw.plugin.CallRender(proxy); err != nil {
		msg := "Plugin error: " + err.Error()
		surface.DrawText(0, 0, msg, w, term.StyleDanger)
	}
}

func (pw *PluginPanelWidget) HandleEvent(ev tcell.Event) widgets.EventResult {
	return widgets.EventIgnored
}

func (pw *PluginPanelWidget) Focusable() bool { return true }
