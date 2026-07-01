package ui

import (
	"fmt"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v2"
)

type OutputLine struct {
	Time       string
	PluginName string
	Level      string
	Message    string
}

type OutputWidget struct {
	Lines      []OutputLine
	autoScroll bool
	tree       *widgets.TreeWidget
}

func NewOutputWidget() *OutputWidget {
	o := &OutputWidget{autoScroll: true}
	o.tree = widgets.NewListWidgetFromConfig(widgets.ListConfig{
		EmptyText: "No output",
		RenderItem: func(surface widgets.Surface, node *widgets.TreeNode, idx, y, w int, selected bool) {
			o.renderItem(surface, idx, y, w, selected)
		},
	})
	return o
}

func (o *OutputWidget) Focusable() bool                 { return true }
func (o *OutputWidget) SetFocused(f bool)               { o.tree.SetFocused(f) }
func (o *OutputWidget) IsFocused() bool                 { return o.tree.IsFocused() }
func (o *OutputWidget) GetRect() Rect                   { return Rect(o.tree.GetRect()) }
func (o *OutputWidget) SetRect(r Rect)                  { o.tree.SetRect(widgets.Rect(r)) }
func (o *OutputWidget) Height() int                     { return 0 }
func (o *OutputWidget) Width() int                      { return 0 }
func (o *OutputWidget) SetBoxModel(bm widgets.BoxModel) { o.tree.SetBoxModel(bm) }
func (o *OutputWidget) Render(surface Surface)          { o.tree.Render(surface) }

func (o *OutputWidget) HandleEvent(ev tcell.Event) EventResult {
	prevSel := o.tree.SelectedIndex()
	result := EventResult(o.tree.HandleEvent(ev))
	if result != EventIgnored && o.tree.SelectedIndex() != prevSel {
		if o.tree.SelectedIndex() == o.tree.ItemCount()-1 {
			o.autoScroll = true
		} else {
			o.autoScroll = false
		}
	}
	return result
}

func (o *OutputWidget) AddLine(line OutputLine) {
	o.Lines = append(o.Lines, line)
	nodes := make([]*widgets.TreeNode, len(o.Lines))
	for i, l := range o.Lines {
		nodes[i] = &widgets.TreeNode{
			ID:    fmt.Sprintf("output-%d", i),
			Label: l.Message,
		}
	}
	o.tree.SetItems(nodes)
	if o.autoScroll {
		o.tree.SetSelectedIndex(len(o.Lines) - 1)
	}
}

func (o *OutputWidget) Clear() {
	o.Lines = nil
	o.tree.SetItems(nil)
	o.autoScroll = true
}

func (o *OutputWidget) renderItem(surface widgets.Surface, idx, y, w int, selected bool) {
	if idx >= len(o.Lines) {
		return
	}
	line := o.Lines[idx]

	style := term.StyleDefault
	if selected {
		style = term.StyleSidebarSelected
	}

	for x := 0; x < w; x++ {
		surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
	}

	x := 1

	levelStyle := style
	switch line.Level {
	case "error":
		levelStyle = term.StyleDanger
	case "warn":
		levelStyle = term.StyleWarning
	}

	prefix := fmt.Sprintf("%s [%s]", line.Time, line.PluginName)
	for _, ch := range prefix {
		if x >= w {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: term.StyleMuted})
		x++
	}
	x++

	for _, ch := range line.Message {
		if x >= w {
			break
		}
		surface.SetCell(x, y, term.Cell{Ch: ch, Style: levelStyle})
		x++
	}
}
