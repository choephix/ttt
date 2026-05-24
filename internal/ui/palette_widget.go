package ui

import (
	"os"
	"path/filepath"
	"strings"
	"ttt/internal/command"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type paletteMode int

const (
	paletteCommandMode paletteMode = iota
	paletteFileMode
)

type PaletteItem struct {
	Label    string
	Detail   string
	ID       string
}

type CommandPaletteWidget struct {
	BaseWidget
	Commands     []command.Command
	Items        []PaletteItem
	Input        *InputWidget
	Selected     int
	scrollOffset int
	inputX       int
	inputY       int
	mode         paletteMode
	files        []string
	OnExecute          func(id string)
	OnOpenFile         func(path string)
	OnDismiss          func()
	OnSelectionChange  func(id string)
	Borders            *term.BorderSet
}

func NewCommandPaletteWidget(commands []command.Command) *CommandPaletteWidget {
	p := &CommandPaletteWidget{
		Commands: commands,
	}
	p.Input = NewInputWidget(" ")
	p.Input.SetText(">")
	p.Input.OnChange = func(text string) {
		p.filter()
	}
	p.filter()
	return p
}

func (p *CommandPaletteWidget) SetFiles(workDir string) {
	var files []string
	filepath.WalkDir(workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == ".cache" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(workDir, path)
		if err != nil {
			return nil
		}
		files = append(files, rel)
		if len(files) >= 10000 {
			return filepath.SkipAll
		}
		return nil
	})
	p.files = files
}

func (p *CommandPaletteWidget) Focusable() bool { return true }

func (p *CommandPaletteWidget) CursorPosition() (int, int, bool) {
	return p.Input.CursorX(p.inputX), p.inputY, true
}

func (p *CommandPaletteWidget) Render(surface *RenderSurface) {
	sw, sh := surface.Size()

	boxW := 60
	if boxW > sw-4 {
		boxW = sw - 4
	}
	maxItems := 10
	boxH := 4 + len(p.Items)
	if boxH > maxItems+4 {
		boxH = maxItems + 4
	}
	if boxH > sh-2 {
		boxH = sh - 2
	}

	boxX := (sw - boxW) / 2
	boxY := 2

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

	for y := boxY + 1; y < boxY+boxH-1; y++ {
		for x := boxX + 1; x < boxX+boxW-1; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' '})
		}
	}

	p.inputX = boxX + 1
	p.inputY = boxY + 1
	p.Input.Render(surface, p.inputX, p.inputY, boxW-2)

	for x := boxX + 1; x < boxX+boxW-1; x++ {
		surface.SetCell(x, boxY+2, term.Cell{Ch: b.Horizontal, Style: term.StyleBorder})
	}

	visibleItems := boxH - 4
	p.ensureVisible(visibleItems)
	showScroll := len(p.Items) > visibleItems
	contentRight := boxX + boxW - 1
	if showScroll {
		contentRight--
	}

	var thumbTop, thumbH int
	if showScroll {
		sb := Scrollbar{Height: visibleItems, TotalItems: len(p.Items), TopItem: p.scrollOffset}
		thumbTop, thumbH = sb.ThumbPos()
	}

	for i := 0; i < visibleItems && p.scrollOffset+i < len(p.Items); i++ {
		y := boxY + 3 + i
		idx := p.scrollOffset + i
		item := p.Items[idx]

		style := term.StylePaletteItem
		if idx == p.Selected {
			style = term.StylePaletteSelected
		}

		for x := boxX + 1; x < contentRight; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		for j, ch := range item.Label {
			x := boxX + 2 + j
			if x < contentRight-1 {
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			}
		}

		if item.Detail != "" {
			detailStyle := term.StyleMuted
			if idx == p.Selected {
				detailStyle = style
			}
			detailRunes := []rune(item.Detail)
			sx := contentRight - 1 - len(detailRunes)
			for j, ch := range detailRunes {
				if sx+j > boxX+1 {
					surface.SetCell(sx+j, y, term.Cell{Ch: ch, Style: detailStyle})
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
		if p.Selected >= 0 && p.Selected < len(p.Items) {
			item := p.Items[p.Selected]
			if p.mode == paletteCommandMode {
				if p.OnExecute != nil {
					p.OnExecute(item.ID)
				}
			} else {
				if p.OnOpenFile != nil {
					p.OnOpenFile(item.ID)
				}
			}
		}
	case tcell.KeyUp:
		if p.Selected > 0 {
			p.Selected--
		} else if len(p.Items) > 0 {
			p.Selected = len(p.Items) - 1
		}
		p.notifySelectionChange()
	case tcell.KeyDown:
		if p.Selected < len(p.Items)-1 {
			p.Selected++
		} else {
			p.Selected = 0
		}
		p.notifySelectionChange()
	default:
		p.Input.HandleEvent(ev)
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
	if p.OnSelectionChange != nil && p.Selected >= 0 && p.Selected < len(p.Items) {
		p.OnSelectionChange(p.Items[p.Selected].ID)
	}
}

func (p *CommandPaletteWidget) filter() {
	text := p.Input.Text
	if strings.HasPrefix(text, ">") {
		p.mode = paletteCommandMode
		query := strings.TrimLeft(text[1:], " ")
		p.filterCommands(query)
	} else {
		p.mode = paletteFileMode
		p.filterFiles(text)
	}
	p.notifySelectionChange()
}

func (p *CommandPaletteWidget) filterCommands(query string) {
	p.Items = nil
	if query == "" {
		for _, cmd := range p.Commands {
			p.Items = append(p.Items, PaletteItem{
				Label:  cmd.Title,
				Detail: cmd.Shortcut,
				ID:     cmd.ID,
			})
		}
	} else {
		lower := strings.ToLower(query)
		for _, cmd := range p.Commands {
			if strings.Contains(strings.ToLower(cmd.Title), lower) {
				p.Items = append(p.Items, PaletteItem{
					Label:  cmd.Title,
					Detail: cmd.Shortcut,
					ID:     cmd.ID,
				})
			}
		}
	}
	p.Selected = 0
	p.scrollOffset = 0
}

func fileDetail(f string) string {
	dir := filepath.Dir(f)
	if dir == "." {
		return ""
	}
	return dir
}

func (p *CommandPaletteWidget) filterFiles(query string) {
	p.Items = nil
	if query == "" {
		for _, f := range p.files {
			p.Items = append(p.Items, PaletteItem{
				Label:  filepath.Base(f),
				Detail: fileDetail(f),
				ID:     f,
			})
			if len(p.Items) >= 100 {
				break
			}
		}
	} else {
		lower := strings.ToLower(query)
		for _, f := range p.files {
			if strings.Contains(strings.ToLower(f), lower) {
				p.Items = append(p.Items, PaletteItem{
					Label:  filepath.Base(f),
					Detail: fileDetail(f),
					ID:     f,
				})
				if len(p.Items) >= 100 {
					break
				}
			}
		}
	}
	p.Selected = 0
	p.scrollOffset = 0
}
