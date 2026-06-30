package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TableColumn struct {
	Label string
	Width int
	Align string // "left" (default), "right", "center"
}

type TableConfig struct {
	Columns     []TableColumn
	Rows        [][]string
	OnSelect    func(rowIndex int)
	OnCommand   func(command string, rowIndex int)
	NodeMenu    []MenuEntry
	KeyCommands map[rune]string
}

type TableWidget struct {
	BaseWidget
	Config    TableConfig
	selected  int
	scrollTop int
	lastSel   int
	focused   bool

	scrollbar scrollbar
	contentW  int
}

func NewTableWidget(cfg TableConfig) *TableWidget {
	return &TableWidget{Config: cfg}
}

func (t *TableWidget) Height() int        { return 0 }
func (t *TableWidget) Width() int         { return 0 }
func (t *TableWidget) Focusable() bool    { return true }
func (t *TableWidget) SetFocused(f bool)  { t.focused = f }
func (t *TableWidget) IsFocused() bool    { return t.focused }
func (t *TableWidget) SelectedIndex() int { return t.selected }

func (t *TableWidget) SetSelectedIndex(i int) {
	t.selected = i
	t.clampSelected()
}

func (t *TableWidget) clampSelected() {
	if t.selected >= len(t.Config.Rows) {
		t.selected = len(t.Config.Rows) - 1
	}
	if t.selected < 0 {
		t.selected = 0
	}
}

func (t *TableWidget) ensureVisible(visibleH int) {
	if t.selected != t.lastSel {
		t.lastSel = t.selected
		if t.selected < t.scrollTop {
			t.scrollTop = t.selected
		}
		if t.selected >= t.scrollTop+visibleH {
			t.scrollTop = t.selected - visibleH + 1
		}
	}
}

func (t *TableWidget) Render(surface Surface) {
	surface = t.RenderBox(surface)
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	if h <= 0 || w <= 0 || len(t.Config.Columns) == 0 {
		return
	}

	// Header takes 2 lines (labels + separator)
	headerH := 2
	dataH := h - headerH
	if dataH < 0 {
		dataH = 0
	}

	maxScroll := len(t.Config.Rows) - dataH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if t.scrollTop > maxScroll {
		t.scrollTop = maxScroll
	}

	t.ensureVisible(dataH)

	t.scrollbar.X = t.rect.X + w - 1
	t.scrollbar.Y = t.rect.Y + headerH
	t.scrollbar.Height = dataH
	t.scrollbar.TotalItems = len(t.Config.Rows)
	t.scrollbar.TopItem = t.scrollTop

	t.contentW = w
	if t.scrollbar.visible() {
		t.contentW = w - 1
	}

	t.renderHeader(surface, t.contentW)

	for i := range dataH {
		idx := t.scrollTop + i
		if idx >= len(t.Config.Rows) {
			break
		}
		t.renderRow(surface, idx, headerH+i, t.contentW)
	}

	t.scrollbar.Render(surface, w-1, headerH)
}

func (t *TableWidget) renderHeader(surface Surface, w int) {
	style := term.StyleHoverBold
	x := 0
	for i, col := range t.Config.Columns {
		if i > 0 {
			if x < w {
				surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
				x++
			}
			if x < w {
				surface.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
				x++
			}
		}
		t.renderCell(surface, col.Label, col, x, 0, style)
		x += col.Width
	}
	// Fill remaining header space
	for fx := x; fx < w; fx++ {
		surface.SetCell(fx, 0, term.Cell{Ch: ' ', Style: style})
	}

	// Separator line
	sepStyle := term.StyleBorder
	for sx := 0; sx < w; sx++ {
		surface.SetCell(sx, 1, term.Cell{Ch: '─', Style: sepStyle})
	}
}

func (t *TableWidget) renderRow(surface Surface, idx, y, w int) {
	style := term.StyleDefault
	if idx == t.selected && t.focused {
		style = term.StyleSidebarSelected
	}

	for fx := 0; fx < w; fx++ {
		surface.SetCell(fx, y, term.Cell{Ch: ' ', Style: style})
	}

	row := t.Config.Rows[idx]
	x := 0
	for i, col := range t.Config.Columns {
		if i > 0 {
			if x < w {
				surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
				x++
			}
			if x < w {
				surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
				x++
			}
		}
		cellText := ""
		if i < len(row) {
			cellText = row[i]
		}
		t.renderCell(surface, cellText, col, x, y, style)
		x += col.Width
	}
}

func (t *TableWidget) renderCell(surface Surface, text string, col TableColumn, x, y int, style term.Style) {
	runes := []rune(text)
	colW := col.Width
	if colW <= 0 {
		return
	}

	if len(runes) > colW {
		// Truncate with ellipsis
		for i := 0; i < colW-1 && x+i < t.contentW; i++ {
			surface.SetCell(x+i, y, term.Cell{Ch: runes[i], Style: style})
		}
		if x+colW-1 < t.contentW {
			surface.SetCell(x+colW-1, y, term.Cell{Ch: '…', Style: style})
		}
		return
	}

	padding := colW - len(runes)
	switch col.Align {
	case "right":
		for i := 0; i < padding && x+i < t.contentW; i++ {
			surface.SetCell(x+i, y, term.Cell{Ch: ' ', Style: style})
		}
		for i, ch := range runes {
			px := x + padding + i
			if px < t.contentW {
				surface.SetCell(px, y, term.Cell{Ch: ch, Style: style})
			}
		}
	case "center":
		leftPad := padding / 2
		for i := 0; i < leftPad && x+i < t.contentW; i++ {
			surface.SetCell(x+i, y, term.Cell{Ch: ' ', Style: style})
		}
		for i, ch := range runes {
			px := x + leftPad + i
			if px < t.contentW {
				surface.SetCell(px, y, term.Cell{Ch: ch, Style: style})
			}
		}
	default: // left
		for i, ch := range runes {
			if x+i < t.contentW {
				surface.SetCell(x+i, y, term.Cell{Ch: ch, Style: style})
			}
		}
	}
}

func (t *TableWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := t.scrollbar.HandleEvent(ev); consumed {
		t.scrollTop = newTop
		return EventConsumed
	}

	prev := t.selected
	var result EventResult
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		result = t.handleMouse(tev)
	case *tcell.EventKey:
		result = t.handleKey(tev)
	default:
		return EventIgnored
	}
	if t.selected != prev && t.Config.OnSelect != nil {
		t.Config.OnSelect(t.selected)
	}
	return result
}

func (t *TableWidget) handleMouse(ev *tcell.EventMouse) EventResult {
	btn := ev.Buttons()
	mx, my := ev.Position()
	r := t.rect

	if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
		return EventIgnored
	}

	headerH := 2

	if btn&tcell.WheelUp != 0 {
		t.scrollTop -= 3
		if t.scrollTop < 0 {
			t.scrollTop = 0
		}
		return EventConsumed
	}
	if btn&tcell.WheelDown != 0 {
		dataH := r.H - headerH
		if dataH < 0 {
			dataH = 0
		}
		max := len(t.Config.Rows) - dataH
		if max < 0 {
			max = 0
		}
		t.scrollTop += 3
		if t.scrollTop > max {
			t.scrollTop = max
		}
		return EventConsumed
	}

	localY := my - r.Y
	if localY < headerH {
		return EventIgnored
	}

	idx := t.scrollTop + (localY - headerH)
	if idx < 0 || idx >= len(t.Config.Rows) {
		return EventIgnored
	}

	if btn&tcell.Button2 != 0 {
		t.selected = idx
		if t.Config.OnCommand != nil && len(t.Config.NodeMenu) > 0 {
			// Right-click fires the first menu entry's command as a convention,
			// but typically this is handled via the context menu system.
		}
		return EventConsumed
	}

	if btn&tcell.Button1 != 0 {
		t.selected = idx
		if t.Config.OnSelect != nil {
			t.Config.OnSelect(idx)
		}
		return EventConsumed
	}

	return EventIgnored
}

func (t *TableWidget) handleKey(ev *tcell.EventKey) EventResult {
	switch ev.Key() {
	case tcell.KeyUp:
		if t.selected > 0 {
			t.selected--
		}
		return EventConsumed
	case tcell.KeyDown:
		if t.selected < len(t.Config.Rows)-1 {
			t.selected++
		}
		return EventConsumed
	case tcell.KeyEnter:
		if t.Config.OnSelect != nil && t.selected >= 0 && t.selected < len(t.Config.Rows) {
			t.Config.OnSelect(t.selected)
		}
		return EventConsumed
	case tcell.KeyRune:
		if t.handleShortcutKey(ev.Rune()) == EventConsumed {
			return EventConsumed
		}
	}
	return EventIgnored
}

func (t *TableWidget) handleShortcutKey(r rune) EventResult {
	if t.Config.KeyCommands == nil {
		return EventIgnored
	}
	cmd, ok := t.Config.KeyCommands[r]
	if !ok {
		return EventIgnored
	}
	if t.Config.OnCommand != nil && t.selected >= 0 && t.selected < len(t.Config.Rows) {
		t.Config.OnCommand(cmd, t.selected)
	}
	return EventConsumed
}
