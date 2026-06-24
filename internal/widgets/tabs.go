package widgets

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/gdamore/tcell/v2"
)

type TabItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Active bool   `json:"-"`
}

type TabsConfig struct {
	Items      []TabItem `json:"items"`
	Style      term.Style `json:"-"`
	OnTabClick func(index int)
	OnOverflow func(screenX, screenY int)
}

type TabsWidget struct {
	BaseWidget
	Config     TabsConfig
	tabSpans   [][2]int
	overSpan   [2]int
	hiddenTabs []int
	wasPressed bool
	focused    bool
	selected   int
}

func NewTabsWidget(config TabsConfig) *TabsWidget {
	return &TabsWidget{Config: config}
}

func (t *TabsWidget) Height() int { return 1 + t.BoxOverheadH() }
func (t *TabsWidget) Width() int  { return 0 }

func (t *TabsWidget) Focusable() bool { return true }
func (t *TabsWidget) SetFocused(f bool) {
	t.focused = f
	if f {
		t.selected = t.activeIndex()
	}
}
func (t *TabsWidget) IsFocused() bool { return t.focused }

func (t *TabsWidget) SetActive(id string) {
	for i := range t.Config.Items {
		t.Config.Items[i].Active = id == t.Config.Items[i].ID
	}
}

func (t *TabsWidget) ActiveID() string {
	for _, item := range t.Config.Items {
		if item.Active {
			return item.ID
		}
	}
	return ""
}

func (t *TabsWidget) Render(surface Surface) {
	inner := t.RenderBox(surface)
	w, _ := inner.Size()
	if w <= 0 {
		return
	}

	for x := range w {
		inner.SetCell(x, 0, term.Cell{Ch: ' '})
	}

	overflowW := 3

	tabWidths := make([]int, len(t.Config.Items))
	total := 0
	for i, item := range t.Config.Items {
		tw := len([]rune(item.Label)) + 2
		tabWidths[i] = tw
		total += tw
	}

	hasOverflow := total > w
	tabAreaW := w
	if hasOverflow {
		tabAreaW -= overflowW
	}

	t.tabSpans = t.tabSpans[:0]
	t.hiddenTabs = t.hiddenTabs[:0]
	x := 0
	for i, item := range t.Config.Items {
		if x+tabWidths[i] > tabAreaW && hasOverflow {
			t.hiddenTabs = append(t.hiddenTabs, i)
			t.tabSpans = append(t.tabSpans, [2]int{0, 0})
			continue
		}
		style := term.StyleInactiveTab
		if item.Active {
			style = term.StyleActiveTab
		}
		ul := t.focused && i == t.selected
		startX := x
		inner.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
		x++
		for _, ch := range item.Label {
			if x >= tabAreaW {
				break
			}
			inner.SetCell(x, 0, term.Cell{Ch: ch, Style: style, Underline: ul})
			x++
		}
		if x < tabAreaW {
			inner.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
			x++
		}
		t.tabSpans = append(t.tabSpans, [2]int{startX, x})
	}

	t.overSpan = [2]int{0, 0}
	if hasOverflow {
		ox := tabAreaW
		t.overSpan = [2]int{ox, ox + overflowW}
		style := term.StyleInactiveTab
		inner.SetCell(ox, 0, term.Cell{Ch: ' ', Style: style})
		inner.SetCell(ox+1, 0, term.Cell{Ch: '»', Style: style})
		inner.SetCell(ox+2, 0, term.Cell{Ch: ' ', Style: style})
	}
}

func (t *TabsWidget) HandleEvent(ev tcell.Event) bool {
	switch tev := ev.(type) {
	case *tcell.EventKey:
		return t.handleKey(tev)
	case *tcell.EventMouse:
		return t.handleMouse(tev)
	}
	return false
}

func (t *TabsWidget) activeIndex() int {
	for i, item := range t.Config.Items {
		if item.Active {
			return i
		}
	}
	return 0
}

func (t *TabsWidget) handleKey(ev *tcell.EventKey) bool {
	if !t.focused {
		return false
	}
	n := len(t.Config.Items)
	if n == 0 {
		return false
	}
	switch ev.Key() {
	case tcell.KeyLeft:
		t.selected--
		if t.selected < 0 {
			t.selected = n - 1
		}
		return true
	case tcell.KeyRight:
		t.selected = (t.selected + 1) % n
		return true
	case tcell.KeyEnter:
		if t.Config.OnTabClick != nil {
			t.Config.OnTabClick(t.selected)
		}
		return true
	case tcell.KeyRune:
		if ev.Rune() == ' ' {
			if t.Config.OnTabClick != nil {
				t.Config.OnTabClick(t.selected)
			}
			return true
		}
	}
	return false
}

func (t *TabsWidget) handleMouse(mev *tcell.EventMouse) bool {
	pressed := mev.Buttons()&tcell.Button1 != 0
	freshClick := pressed && !t.wasPressed
	t.wasPressed = pressed
	if !freshClick {
		return false
	}
	mx, my := mev.Position()
	r := t.GetRect()
	if my < r.Y || my >= r.Y+r.H || mx < r.X || mx >= r.X+r.W {
		return false
	}
	lx := mx - r.X - t.Box.MarginLeft - t.Box.PaddingLeft

	if t.Config.OnOverflow != nil && t.overSpan[1] > 0 && lx >= t.overSpan[0] && lx < t.overSpan[1] {
		t.Config.OnOverflow(mx, my)
		return true
	}

	for i, span := range t.tabSpans {
		if lx >= span[0] && lx < span[1] {
			if t.Config.OnTabClick != nil {
				t.Config.OnTabClick(i)
			}
			return true
		}
	}
	return false
}
