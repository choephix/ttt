package ui

import (
	"ttt/internal/command"
	"ttt/internal/term"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type CommandPaletteWidget struct {
	BaseWidget
	Commands     []command.Command
	Filtered     []command.Command
	Query        string
	Selected     int
	scrollOffset int
	OnExecute          func(id string)
	OnDismiss          func()
	OnSelectionChange  func(id string)
	Borders            *term.BorderSet
}

func NewCommandPaletteWidget(commands []command.Command) *CommandPaletteWidget {
	p := &CommandPaletteWidget{
		Commands: commands,
	}
	p.filterCommands()
	return p
}

func (p *CommandPaletteWidget) Focusable() bool { return true }

func (p *CommandPaletteWidget) Render(surface *RenderSurface) {
	sw, sh := surface.Size()

	// Centered box
	boxW := 60
	if boxW > sw-4 {
		boxW = sw - 4
	}
	maxItems := 10
	boxH := 4 + len(p.Filtered)
	if boxH > maxItems+4 {
		boxH = maxItems + 4
	}
	if boxH > sh-2 {
		boxH = sh - 2
	}

	boxX := (sw - boxW) / 2
	boxY := 2

	// Draw border
	b := term.DoubleBorderSet()
	if p.Borders != nil {
		b = *p.Borders
	}
	for x := boxX; x < boxX+boxW; x++ {
		surface.SetCell(x, boxY, term.Cell{Ch: b.Horizontal, Style: term.StyleBorder})
		surface.SetCell(x, boxY+boxH-1, term.Cell{Ch: b.Horizontal, Style: term.StyleBorder})
	}
	for y := boxY; y < boxY+boxH; y++ {
		surface.SetCell(boxX, y, term.Cell{Ch: b.Vertical, Style: term.StyleBorder})
		surface.SetCell(boxX+boxW-1, y, term.Cell{Ch: b.Vertical, Style: term.StyleBorder})
	}
	surface.SetCell(boxX, boxY, term.Cell{Ch: b.TopLeft, Style: term.StyleBorder})
	surface.SetCell(boxX+boxW-1, boxY, term.Cell{Ch: b.TopRight, Style: term.StyleBorder})
	surface.SetCell(boxX, boxY+boxH-1, term.Cell{Ch: b.BottomLeft, Style: term.StyleBorder})
	surface.SetCell(boxX+boxW-1, boxY+boxH-1, term.Cell{Ch: b.BottomRight, Style: term.StyleBorder})

	// Clear interior
	for y := boxY + 1; y < boxY+boxH-1; y++ {
		for x := boxX + 1; x < boxX+boxW-1; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' '})
		}
	}

	// Input line
	inputLine := " > " + p.Query
	for i, ch := range inputLine {
		x := boxX + 1 + i
		if x < boxX+boxW-1 {
			surface.SetCell(x, boxY+1, term.Cell{Ch: ch, Style: term.StyleDefault})
		}
	}

	// Separator
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, boxY+2, term.Cell{Ch: b.Horizontal, Style: term.StyleBorder})
	}

	// Command list
	visibleItems := boxH - 4
	p.ensureVisible(visibleItems)
	showScroll := len(p.Filtered) > visibleItems
	contentRight := boxX + boxW - 1
	if showScroll {
		contentRight--
	}

	var thumbTop, thumbH int
	if showScroll {
		thumbTop, thumbH = scrollbarThumb(len(p.Filtered), p.scrollOffset, visibleItems)
	}

	for i := 0; i < visibleItems && p.scrollOffset+i < len(p.Filtered); i++ {
		y := boxY + 3 + i
		idx := p.scrollOffset + i
		cmd := p.Filtered[idx]

		style := term.StylePaletteItem
		if idx == p.Selected {
			style = term.StylePaletteSelected
		}

		for x := boxX + 1; x < contentRight; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		for j, ch := range cmd.Title {
			x := boxX + 2 + j
			if x < contentRight-1 {
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			}
		}

		if cmd.Shortcut != "" {
			shortStyle := term.StyleMuted
			if idx == p.Selected {
				shortStyle = style
			}
			shortRunes := []rune(cmd.Shortcut)
			sx := contentRight - 1 - len(shortRunes)
			for j, ch := range shortRunes {
				if sx+j > boxX+1 {
					surface.SetCell(sx+j, y, term.Cell{Ch: ch, Style: shortStyle})
				}
			}
		}

		if showScroll {
			sx := boxX + boxW - 2
			if i >= thumbTop && i < thumbTop+thumbH {
				surface.SetCell(sx, y, term.Cell{Ch: '█', Style: term.StyleScrollbarThumb})
			} else {
				surface.SetCell(sx, y, term.Cell{Ch: ' ', Style: term.StyleScrollbar})
			}
		}
	}
}

func (p *CommandPaletteWidget) HandleEvent(ev tcell.Event) EventResult {
	kev, ok := ev.(*tcell.EventKey)
	if !ok {
		return EventConsumed
	}

	switch kev.Key() {
	case tcell.KeyEscape:
		if p.OnDismiss != nil {
			p.OnDismiss()
		}
	case tcell.KeyEnter:
		if p.Selected >= 0 && p.Selected < len(p.Filtered) {
			if p.OnExecute != nil {
				p.OnExecute(p.Filtered[p.Selected].ID)
			}
		}
	case tcell.KeyUp:
		if p.Selected > 0 {
			p.Selected--
		} else if len(p.Filtered) > 0 {
			p.Selected = len(p.Filtered) - 1
		}
		p.notifySelectionChange()
	case tcell.KeyDown:
		if p.Selected < len(p.Filtered)-1 {
			p.Selected++
		} else {
			p.Selected = 0
		}
		p.notifySelectionChange()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(p.Query) > 0 {
			runes := []rune(p.Query)
			p.Query = string(runes[:len(runes)-1])
			p.filterCommands()
			p.notifySelectionChange()
		}
	case tcell.KeyRune:
		p.Query += string(kev.Rune())
		p.filterCommands()
		p.notifySelectionChange()
	}

	return EventConsumed
}

func (p *CommandPaletteWidget) ensureVisible(visibleItems int) {
	if visibleItems <= 0 {
		return
	}
	if p.Selected < p.scrollOffset {
		p.scrollOffset = p.Selected
	}
	if p.Selected >= p.scrollOffset+visibleItems {
		p.scrollOffset = p.Selected - visibleItems + 1
	}
}

func (p *CommandPaletteWidget) notifySelectionChange() {
	if p.OnSelectionChange != nil && p.Selected >= 0 && p.Selected < len(p.Filtered) {
		p.OnSelectionChange(p.Filtered[p.Selected].ID)
	}
}

func (p *CommandPaletteWidget) filterCommands() {
	if p.Query == "" {
		p.Filtered = p.Commands
	} else {
		p.Filtered = nil
		lower := strings.ToLower(p.Query)
		for _, cmd := range p.Commands {
			if strings.Contains(strings.ToLower(cmd.Title), lower) {
				p.Filtered = append(p.Filtered, cmd)
			}
		}
	}
	p.Selected = 0
	p.scrollOffset = 0
}
