package ui

import (
	"macro/internal/command"
	"macro/internal/term"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type CommandPaletteWidget struct {
	BaseWidget
	Commands  []command.Command
	Filtered  []command.Command
	Query     string
	Selected  int
	OnExecute func(id string)
	OnDismiss func()
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
	boxW := 50
	if boxW > sw-4 {
		boxW = sw - 4
	}
	maxItems := 10
	boxH := 3 + len(p.Filtered)
	if boxH > maxItems+3 {
		boxH = maxItems + 3
	}
	if boxH > sh-2 {
		boxH = sh - 2
	}

	boxX := (sw - boxW) / 2
	boxY := 2

	// Draw border
	for x := boxX; x < boxX+boxW; x++ {
		surface.SetCell(x, boxY, term.Cell{Ch: '─', Style: term.StylePaletteBorder})
		surface.SetCell(x, boxY+boxH-1, term.Cell{Ch: '─', Style: term.StylePaletteBorder})
	}
	for y := boxY; y < boxY+boxH; y++ {
		surface.SetCell(boxX, y, term.Cell{Ch: '│', Style: term.StylePaletteBorder})
		surface.SetCell(boxX+boxW-1, y, term.Cell{Ch: '│', Style: term.StylePaletteBorder})
	}
	surface.SetCell(boxX, boxY, term.Cell{Ch: '┌', Style: term.StylePaletteBorder})
	surface.SetCell(boxX+boxW-1, boxY, term.Cell{Ch: '┐', Style: term.StylePaletteBorder})
	surface.SetCell(boxX, boxY+boxH-1, term.Cell{Ch: '└', Style: term.StylePaletteBorder})
	surface.SetCell(boxX+boxW-1, boxY+boxH-1, term.Cell{Ch: '┘', Style: term.StylePaletteBorder})

	// Clear interior
	for y := boxY + 1; y < boxY+boxH-1; y++ {
		for x := boxX + 1; x < boxX+boxW-1; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' '})
		}
	}

	// Input line
	inputLine := "> " + p.Query
	for i, ch := range inputLine {
		x := boxX + 1 + i
		if x < boxX+boxW-1 {
			surface.SetCell(x, boxY+1, term.Cell{Ch: ch, Style: term.StylePaletteInput})
		}
	}

	// Separator
	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, boxY+2, term.Cell{Ch: '─', Style: term.StylePaletteBorder})
	}

	// Command list
	visibleItems := boxH - 3
	for i := 0; i < visibleItems && i < len(p.Filtered); i++ {
		y := boxY + 3 + i
		cmd := p.Filtered[i]

		style := term.StylePaletteItem
		if i == p.Selected {
			style = term.StylePaletteSelected
		}

		// Fill background
		for x := boxX + 1; x < boxX+boxW-1; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		// Command title
		for j, ch := range cmd.Title {
			x := boxX + 2 + j
			if x < boxX+boxW-2 {
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
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
		}
	case tcell.KeyDown:
		if p.Selected < len(p.Filtered)-1 {
			p.Selected++
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(p.Query) > 0 {
			runes := []rune(p.Query)
			p.Query = string(runes[:len(runes)-1])
			p.filterCommands()
		}
	case tcell.KeyRune:
		p.Query += string(kev.Rune())
		p.filterCommands()
	}

	return EventConsumed
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
}
