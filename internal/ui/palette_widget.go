package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type paletteMode int

const (
	paletteCommandMode paletteMode = iota
	paletteFileMode
	paletteGoToLineMode
)

type PaletteItem struct {
	Label  string
	Detail string
	ID     string
}

type paletteFile struct {
	Rel string
	Abs string
}

type CommandPaletteWidget struct {
	BaseWidget
	Commands          []command.Command
	Items             []PaletteItem
	Input             *InputWidget
	Selected          int
	scrollOffset      int
	inputX            int
	inputY            int
	mode              paletteMode
	files             []paletteFile
	OnExecute         func(id string)
	OnOpenFile        func(path string)
	OnGoToLine        func(line int)
	OnDismiss         func()
	OnSelectionChange func(id string)
	Borders           *term.BorderSet
}

func NewCommandPaletteWidget(commands []command.Command) *CommandPaletteWidget {
	p := &CommandPaletteWidget{
		Commands: commands,
	}
	p.Input = NewInputWidget()
	p.Input.Prefix = " "
	p.Input.SetText(">")
	p.Input.OnChange = func(text string) {
		p.filter()
	}
	p.filter()
	return p
}

func (p *CommandPaletteWidget) SetFiles(workDirs []string) {
	p.files = nil
	multiRoot := len(workDirs) > 1
	for _, workDir := range workDirs {
		prefix := ""
		if multiRoot {
			prefix = filepath.Base(workDir) + string(filepath.Separator)
		}
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
			p.files = append(p.files, paletteFile{Rel: prefix + rel, Abs: path})
			if len(p.files) >= 10000 {
				return filepath.SkipAll
			}
			return nil
		})
	}
}

func (p *CommandPaletteWidget) Focusable() bool { return true }

func (p *CommandPaletteWidget) CursorPosition() (int, int, bool) {
	return p.Input.CursorX(p.inputX), p.inputY, true
}

func (p *CommandPaletteWidget) Render(surface *RenderSurface) {
	sw, sh := surface.Size()

	boxW := sw * 6 / 10 // 60% of terminal width
	if boxW > 80 {
		boxW = 80
	}
	if boxW < 40 {
		boxW = 40
	}
	if boxW > sw-4 {
		boxW = sw - 4
	}
	var boxH int
	if p.mode == paletteGoToLineMode {
		boxH = 3
	} else {
		maxItems := 10
		boxH = 4 + len(p.Items)
		if boxH > maxItems+4 {
			boxH = maxItems + 4
		}
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
	surface.DrawBorder(boxX, boxY, boxW, boxH, b, term.StyleBorder)

	surface.ClearRect(boxX+1, boxY+1, boxW-2, boxH-2, term.StyleDefault)

	p.inputX = boxX + 1
	p.inputY = boxY + 1
	p.Input.Render(surface, p.inputX, p.inputY, boxW-2)

	if p.mode == paletteGoToLineMode {
		return
	}

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

		surface.ClearRect(boxX+1, y, contentRight-boxX-1, 1, style)
		surface.DrawText(boxX+2, y, item.Label, contentRight-1, style)

		if item.Detail != "" {
			detailStyle := term.StyleMuted
			if idx == p.Selected {
				detailStyle = style
			}
			detailRunes := []rune(item.Detail)
			sx := contentRight - 1 - len(detailRunes)
			if sx > boxX+1 {
				surface.DrawText(sx, y, item.Detail, contentRight-1, detailStyle)
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
	if mev, ok := ev.(*tcell.EventMouse); ok {
		if mev.Buttons()&tcell.Button1 != 0 {
			mx, my := mev.Position()
			if my == p.inputY {
				p.Input.HandleClick(mx, my)
			}
		}
		return EventConsumed
	}

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
		if p.mode == paletteGoToLineMode {
			if p.OnGoToLine != nil {
				text := strings.TrimPrefix(p.Input.Text, ":")
				if n, err := strconv.Atoi(text); err == nil && n > 0 {
					p.OnGoToLine(n)
				}
			}
		} else if p.Selected >= 0 && p.Selected < len(p.Items) {
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
	} else if strings.HasPrefix(text, ":") {
		p.mode = paletteGoToLineMode
		p.Items = nil
		p.Selected = 0
		p.scrollOffset = 0
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
		type scored struct {
			item  PaletteItem
			score int
		}
		var matches []scored
		for _, cmd := range p.Commands {
			bestOk, bestScore := fuzzyMatch(query, cmd.Title)
			for _, kw := range cmd.Keywords {
				if ok, score := fuzzyMatch(query, kw); ok && (!bestOk || score > bestScore) {
					bestOk = true
					bestScore = score
				}
			}
			if bestOk {
				matches = append(matches, scored{
					item: PaletteItem{
						Label:  cmd.Title,
						Detail: cmd.Shortcut,
						ID:     cmd.ID,
					},
					score: bestScore,
				})
			}
		}
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
		for _, m := range matches {
			p.Items = append(p.Items, m.item)
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
				Label:  filepath.Base(f.Rel),
				Detail: fileDetail(f.Rel),
				ID:     f.Abs,
			})
			if len(p.Items) >= 100 {
				break
			}
		}
	} else {
		type scored struct {
			item  PaletteItem
			score int
		}
		var matches []scored
		for _, f := range p.files {
			if ok, score := fuzzyMatch(query, f.Rel); ok {
				matches = append(matches, scored{
					item: PaletteItem{
						Label:  filepath.Base(f.Rel),
						Detail: fileDetail(f.Rel),
						ID:     f.Abs,
					},
					score: score,
				})
			}
		}
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
		for _, m := range matches {
			p.Items = append(p.Items, m.item)
			if len(p.Items) >= 100 {
				break
			}
		}
	}
	p.Selected = 0
	p.scrollOffset = 0
}
