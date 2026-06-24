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
}

func NewTabsWidget(config TabsConfig) *TabsWidget {
	return &TabsWidget{Config: config}
}

func (t *TabsWidget) Height() int { return 1 + t.BoxOverheadH() }
func (t *TabsWidget) Width() int  { return 0 }

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
		startX := x
		inner.SetCell(x, 0, term.Cell{Ch: ' ', Style: style})
		x++
		for _, ch := range item.Label {
			if x >= tabAreaW {
				break
			}
			inner.SetCell(x, 0, term.Cell{Ch: ch, Style: style})
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
	mev, ok := ev.(*tcell.EventMouse)
	if !ok {
		return false
	}
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
