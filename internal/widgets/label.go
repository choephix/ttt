package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type LabelConfig struct {
	Text  string     `json:"text"`
	Style term.Style `json:"-"`
}

type LabelWidget struct {
	BaseWidget
	Config LabelConfig
}

func NewLabelWidget(config LabelConfig) *LabelWidget {
	return &LabelWidget{Config: config}
}

func (l *LabelWidget) Height() int { return 1 + l.BoxOverheadH() }
func (l *LabelWidget) Width() int  { return 0 }

func (l *LabelWidget) Render(surface Surface) {
	inner := l.RenderBox(surface)
	w, _ := inner.Size()
	if w <= 0 {
		return
	}
	style := l.Config.Style
	if style == 0 {
		style = term.StyleDefault
	}
	inner.DrawText(0, 0, l.Config.Text, w, style)
}

func (l *LabelWidget) HandleEvent(ev tcell.Event) bool {
	return false
}
