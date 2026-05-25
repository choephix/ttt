package ui

import (
	"sort"
	"strings"

	"github.com/eugenioenko/ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type CompletionKind int

const (
	CompletionFunction CompletionKind = iota
	CompletionMethod
	CompletionVariable
	CompletionConstant
	CompletionType
	CompletionField
	CompletionKeyword
	CompletionSnippet
	CompletionModule
)

func (k CompletionKind) Symbol() rune { return '■' }

func (k CompletionKind) Style() term.Style {
	switch k {
	case CompletionFunction:
		return term.StyleSyntaxFunction
	case CompletionMethod:
		return term.StyleSyntaxBuiltin
	case CompletionVariable:
		return term.StyleSyntaxVariable
	case CompletionConstant:
		return term.StyleSyntaxNumber
	case CompletionType:
		return term.StyleSyntaxType
	case CompletionField:
		return term.StyleSyntaxTag
	case CompletionKeyword:
		return term.StyleSyntaxKeyword
	case CompletionSnippet:
		return term.StyleSyntaxString
	case CompletionModule:
		return term.StyleSyntaxComment
	default:
		return term.StyleSyntaxVariable
	}
}

type CompletionItem struct {
	Label      string
	Detail     string
	InsertText string
	FilterText string
	SortText   string
	Kind       CompletionKind
}

func FilterCompletions(items []CompletionItem, prefix string) []CompletionItem {
	if prefix == "" {
		return items
	}
	lowerPrefix := strings.ToLower(prefix)
	var filtered []CompletionItem
	for _, it := range items {
		ft := it.FilterText
		if ft == "" {
			ft = it.Label
		}
		if strings.HasPrefix(strings.ToLower(ft), lowerPrefix) {
			filtered = append(filtered, it)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		si := filtered[i].SortText
		if si == "" {
			si = filtered[i].Label
		}
		sj := filtered[j].SortText
		if sj == "" {
			sj = filtered[j].Label
		}
		return si < sj
	})
	return filtered
}

type AutocompleteWidget struct {
	BaseWidget
	Items      []CompletionItem
	Selected   int
	scrollTop  int
	AnchorX    int
	AnchorY    int
	MaxVisible int
	Borders    *term.BorderSet
	OnSelect   func(item CompletionItem)
	OnDismiss  func()
	firstEvent bool
}

const defaultMaxVisible = 10

func NewAutocompleteWidget(items []CompletionItem, x, y int) *AutocompleteWidget {
	return &AutocompleteWidget{
		Items:      items,
		AnchorX:    x,
		AnchorY:    y,
		MaxVisible: defaultMaxVisible,
		firstEvent: true,
	}
}

func (a *AutocompleteWidget) Focusable() bool { return false }

func (a *AutocompleteWidget) SetItems(items []CompletionItem) {
	a.Items = items
	if a.Selected >= len(items) {
		a.Selected = 0
	}
	a.scrollTop = 0
	a.ensureVisible()
}

func (a *AutocompleteWidget) visibleCount() int {
	if len(a.Items) < a.MaxVisible {
		return len(a.Items)
	}
	return a.MaxVisible
}

func (a *AutocompleteWidget) menuWidth() int {
	maxW := 0
	for _, it := range a.Items {
		// "  ■ label  " = 5 + label length
		w := len([]rune(it.Label)) + 5
		if w > maxW {
			maxW = w
		}
	}
	if maxW < 12 {
		maxW = 12
	}
	return maxW
}

func (a *AutocompleteWidget) ensureVisible() {
	vis := a.visibleCount()
	if vis == 0 {
		return
	}
	if a.Selected < a.scrollTop {
		a.scrollTop = a.Selected
	}
	if a.Selected >= a.scrollTop+vis {
		a.scrollTop = a.Selected - vis + 1
	}
}

func (a *AutocompleteWidget) Render(surface *RenderSurface) {
	if len(a.Items) == 0 {
		return
	}
	sw, sh := surface.Size()

	vis := a.visibleCount()
	menuW := a.menuWidth()
	hasScroll := len(a.Items) > vis
	if hasScroll {
		menuW++
	}
	menuH := vis + 2

	x := a.AnchorX
	if x+menuW > sw {
		x = sw - menuW
	}
	if x < 0 {
		x = 0
	}

	spaceBelow := sh - (a.AnchorY + 1)
	spaceAbove := a.AnchorY

	y := a.AnchorY + 1
	if menuH > spaceBelow && menuH <= spaceAbove {
		y = a.AnchorY - menuH
		if y < 0 {
			y = 0
		}
	}

	b := term.SingleBorderSet()
	if a.Borders != nil {
		b = *a.Borders
	}
	bs := term.StyleBorder

	for bx := x; bx < x+menuW; bx++ {
		surface.SetCell(bx, y, term.Cell{Ch: b.Horizontal, Style: bs})
		surface.SetCell(bx, y+menuH-1, term.Cell{Ch: b.Horizontal, Style: bs})
	}
	for by := y; by < y+menuH; by++ {
		surface.SetCell(x, by, term.Cell{Ch: b.Vertical, Style: bs})
		surface.SetCell(x+menuW-1, by, term.Cell{Ch: b.Vertical, Style: bs})
	}
	surface.SetCell(x, y, term.Cell{Ch: b.TopLeft, Style: bs})
	surface.SetCell(x+menuW-1, y, term.Cell{Ch: b.TopRight, Style: bs})
	surface.SetCell(x, y+menuH-1, term.Cell{Ch: b.BottomLeft, Style: bs})
	surface.SetCell(x+menuW-1, y+menuH-1, term.Cell{Ch: b.BottomRight, Style: bs})

	contentW := menuW - 2
	if hasScroll {
		contentW--
	}

	for i := 0; i < vis; i++ {
		idx := a.scrollTop + i
		if idx >= len(a.Items) {
			break
		}
		it := a.Items[idx]
		row := y + 1 + i

		style := term.StylePaletteItem
		if idx == a.Selected {
			style = term.StylePaletteSelected
		}

		for bx := x + 1; bx < x+menuW-1; bx++ {
			surface.SetCell(bx, row, term.Cell{Ch: ' ', Style: style})
		}

		surface.SetCell(x+2, row, term.Cell{Ch: it.Kind.Symbol(), Style: it.Kind.Style()})

		for j, ch := range []rune(it.Label) {
			cx := x + 4 + j
			if cx >= x+1+contentW {
				break
			}
			surface.SetCell(cx, row, term.Cell{Ch: ch, Style: style})
		}
	}

	if hasScroll {
		sb := Scrollbar{
			X:          x + menuW - 2,
			Y:          y + 1,
			Height:     vis,
			TotalItems: len(a.Items),
			TopItem:    a.scrollTop,
		}
		sb.Render(surface, x+menuW-2, y+1)
	}

	a.storeRect(x, y, menuW, menuH)
}

func (a *AutocompleteWidget) storeRect(x, y, w, h int) {
	a.SetRect(Rect{X: x, Y: y, W: w, H: h})
}

func (a *AutocompleteWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		switch tev.Key() {
		case tcell.KeyEscape:
			if a.OnDismiss != nil {
				a.OnDismiss()
			}
			return EventConsumed
		case tcell.KeyUp:
			a.moveSelection(-1)
			return EventConsumed
		case tcell.KeyDown:
			a.moveSelection(1)
			return EventConsumed
		case tcell.KeyEnter, tcell.KeyTab:
			if a.Selected >= 0 && a.Selected < len(a.Items) {
				if a.OnSelect != nil {
					a.OnSelect(a.Items[a.Selected])
				}
			}
			return EventConsumed
		}
		return EventIgnored

	case *tcell.EventMouse:
		btn := tev.Buttons()
		mx, my := tev.Position()
		r := a.GetRect()

		if a.firstEvent {
			a.firstEvent = false
			return EventConsumed
		}

		if btn&tcell.WheelUp != 0 {
			if a.scrollTop > 0 {
				a.scrollTop--
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			max := len(a.Items) - a.visibleCount()
			if a.scrollTop < max {
				a.scrollTop++
			}
			return EventConsumed
		}

		if btn&tcell.Button1 != 0 {
			if mx < r.X || mx >= r.X+r.W || my < r.Y || my >= r.Y+r.H {
				if a.OnDismiss != nil {
					a.OnDismiss()
				}
				return EventConsumed
			}
			itemIdx := my - r.Y - 1 + a.scrollTop
			if itemIdx >= 0 && itemIdx < len(a.Items) {
				a.Selected = itemIdx
				if a.OnSelect != nil {
					a.OnSelect(a.Items[a.Selected])
				}
			}
			return EventConsumed
		}

		if btn == tcell.ButtonNone {
			itemIdx := my - r.Y - 1 + a.scrollTop
			if itemIdx >= 0 && itemIdx < len(a.Items) {
				a.Selected = itemIdx
			}
			return EventConsumed
		}
	}
	return EventIgnored
}

func (a *AutocompleteWidget) moveSelection(dir int) {
	n := len(a.Items)
	if n == 0 {
		return
	}
	a.Selected += dir
	if a.Selected < 0 {
		a.Selected = 0
	}
	if a.Selected >= n {
		a.Selected = n - 1
	}
	a.ensureVisible()
}
