package ui

import (
	"fmt"
	"path/filepath"

	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type ProblemItem struct {
	File     string
	Line     int
	Col      int
	Severity DiagnosticSeverity
	Message  string
	Source   string
}

type ProblemsWidget struct {
	BaseWidget
	Items      []ProblemItem
	selected   int
	scrollTop  int
	OnNavigate func(file string, line, col int)
}

func NewProblemsWidget() *ProblemsWidget {
	return &ProblemsWidget{}
}

func (p *ProblemsWidget) Focusable() bool { return true }

func (p *ProblemsWidget) SetItems(items []ProblemItem) {
	p.Items = items
	if p.selected >= len(items) {
		p.selected = len(items) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
}

func (p *ProblemsWidget) HasProblems() bool {
	for _, item := range p.Items {
		if item.Severity <= DiagWarning {
			return true
		}
	}
	return false
}

func (p *ProblemsWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()

	if len(p.Items) == 0 {
		msg := "No problems detected"
		x := 1
		for _, ch := range msg {
			if x >= w {
				break
			}
			surface.SetCell(x, 0, term.Cell{Ch: ch, Style: term.StyleMuted})
			x++
		}
		return
	}

	if p.scrollTop > p.selected {
		p.scrollTop = p.selected
	}
	if p.selected >= p.scrollTop+h {
		p.scrollTop = p.selected - h + 1
	}

	for y := 0; y < h; y++ {
		idx := p.scrollTop + y
		if idx >= len(p.Items) {
			break
		}
		item := p.Items[idx]

		style := term.StyleDefault
		if idx == p.selected {
			style = term.StyleSidebarSelected
		}

		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		x := 1
		// Severity icon
		icon := 'E'
		iconStyle := term.StyleDanger
		switch item.Severity {
		case DiagWarning:
			icon = 'W'
			iconStyle = term.StyleWarning
		case DiagInformation:
			icon = 'I'
			iconStyle = term.StyleMuted
		case DiagHint:
			icon = 'H'
			iconStyle = term.StyleMuted
		}
		surface.SetCell(x, y, term.Cell{Ch: icon, Style: iconStyle})
		x += 2

		// File:line
		loc := fmt.Sprintf("%s:%d", filepath.Base(item.File), item.Line+1)
		for _, ch := range loc {
			if x >= w {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			x++
		}
		x++

		// Message
		msgStyle := style
		if idx != p.selected {
			msgStyle = term.StyleMuted
		}
		for _, ch := range item.Message {
			if x >= w-1 {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: msgStyle})
			x++
		}
	}
}

func (p *ProblemsWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		switch tev.Key() {
		case tcell.KeyUp:
			if p.selected > 0 {
				p.selected--
			}
			return EventConsumed
		case tcell.KeyDown:
			if p.selected < len(p.Items)-1 {
				p.selected++
			}
			return EventConsumed
		case tcell.KeyEnter:
			if p.selected < len(p.Items) && p.OnNavigate != nil {
				item := p.Items[p.selected]
				p.OnNavigate(item.File, item.Line, item.Col)
			}
			return EventConsumed
		}
	case *tcell.EventMouse:
		btn := tev.Buttons()
		_, my := tev.Position()
		r := p.GetRect()
		row := my - r.Y
		idx := p.scrollTop + row
		if btn&tcell.Button1 != 0 && idx >= 0 && idx < len(p.Items) {
			p.selected = idx
			if p.OnNavigate != nil {
				item := p.Items[p.selected]
				p.OnNavigate(item.File, item.Line, item.Col)
			}
			return EventConsumed
		}
		if btn&tcell.WheelUp != 0 {
			if p.scrollTop > 0 {
				p.scrollTop--
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			if p.scrollTop < len(p.Items)-1 {
				p.scrollTop++
			}
			return EventConsumed
		}
	}
	return EventIgnored
}
