package ui

import (
	"ttt/internal/git"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ChangesWidget struct {
	BaseWidget
	Dir        string
	Files      []git.FileStatus
	Selected   int
	ScrollTop  int
	OnOpenDiff func(status git.FileStatus)
}

func NewChangesWidget(dir string) *ChangesWidget {
	w := &ChangesWidget{Dir: dir}
	w.Refresh()
	return w
}

func (c *ChangesWidget) Focusable() bool { return true }

func (c *ChangesWidget) Refresh() {
	files, err := git.StatusFiles(c.Dir)
	if err != nil {
		c.Files = nil
		return
	}
	c.Files = files
	if c.Selected >= len(c.Files) {
		c.Selected = len(c.Files) - 1
	}
	if c.Selected < 0 {
		c.Selected = 0
	}
}

func statusStyle(status string) term.Style {
	switch status {
	case "M":
		return term.StyleDiffModified
	case "A", "??":
		return term.StyleDiffAdded
	case "D":
		return term.StyleDiffDeleted
	default:
		return term.StyleSidebarItem
	}
}

func (c *ChangesWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	if len(c.Files) == 0 {
		msg := "No changes"
		for i, ch := range msg {
			if i+1 < w {
				surface.SetCell(i+1, 0, term.Cell{Ch: ch, Style: term.StyleSidebarItem})
			}
		}
		return
	}

	if h <= 0 {
		return
	}
	if c.Selected < c.ScrollTop {
		c.ScrollTop = c.Selected
	}
	if c.Selected >= c.ScrollTop+h {
		c.ScrollTop = c.Selected - h + 1
	}

	for i := 0; i < h; i++ {
		idx := c.ScrollTop + i
		if idx >= len(c.Files) {
			break
		}
		f := c.Files[idx]
		y := i

		style := term.StyleSidebarItem
		if idx == c.Selected {
			style = term.StyleSidebarSelected
		}

		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		x := 1
		badge := statusBadge(f.Status)
		badgeStyle := statusStyle(f.Status)
		if idx == c.Selected {
			badgeStyle = style
		}
		for _, ch := range badge {
			if x < w {
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: badgeStyle})
				x++
			}
		}
		if x < w {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
			x++
		}

		for _, ch := range f.Path {
			if x >= w {
				break
			}
			surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
			x++
		}
	}
}

func statusBadge(status string) string {
	switch status {
	case "M":
		return "M"
	case "A":
		return "A"
	case "D":
		return "D"
	case "R":
		return "R"
	case "??":
		return "U"
	default:
		return status
	}
}

func (c *ChangesWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		if tev.Buttons()&tcell.Button1 != 0 {
			_, my := tev.Position()
			r := c.GetRect()
			localY := my - r.Y
			idx := c.ScrollTop + localY
			if idx >= 0 && idx < len(c.Files) {
				c.Selected = idx
				c.openSelected()
			}
			return EventConsumed
		}
	case *tcell.EventKey:
		switch tev.Key() {
		case tcell.KeyUp:
			if c.Selected > 0 {
				c.Selected--
			}
			return EventConsumed
		case tcell.KeyDown:
			if c.Selected < len(c.Files)-1 {
				c.Selected++
			}
			return EventConsumed
		case tcell.KeyEnter:
			c.openSelected()
			return EventConsumed
		case tcell.KeyRune:
			if tev.Rune() == 'r' || tev.Rune() == 'R' {
				c.Refresh()
				return EventConsumed
			}
		}
	}
	return EventIgnored
}

func (c *ChangesWidget) openSelected() {
	if c.Selected < 0 || c.Selected >= len(c.Files) {
		return
	}
	if c.OnOpenDiff != nil {
		c.OnOpenDiff(c.Files[c.Selected])
	}
}
