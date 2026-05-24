package ui

import (
	"ttt/internal/git"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type ChangesWidget struct {
	BaseWidget
	SelectableList
	Dir        string
	Files      []git.FileStatus
	OnOpenDiff   func(status git.FileStatus)
	OnRightClick func(status git.FileStatus, screenX, screenY int)
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
	c.ClampSelected(len(c.Files))
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
		return term.StyleDefault
	}
}

func (c *ChangesWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	if len(c.Files) == 0 {
		msg := "No changes"
		for i, ch := range msg {
			if i+1 < w {
				surface.SetCell(i+1, 0, term.Cell{Ch: ch, Style: term.StyleDefault})
			}
		}
		return
	}

	if h <= 0 {
		return
	}
	c.EnsureVisible(h)

	for i := 0; i < h; i++ {
		idx := c.ScrollTop + i
		if idx >= len(c.Files) {
			break
		}
		f := c.Files[idx]
		y := i

		style := term.StyleDefault
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
	r := c.GetRect()
	res := c.SelectableList.HandleListEvent(ev, r, len(c.Files))
	if res.Result == EventConsumed {
		switch res.Action {
		case ListActionActivate:
			c.openSelected()
		case ListActionContext:
			if c.OnRightClick != nil && c.Selected >= 0 && c.Selected < len(c.Files) {
				c.OnRightClick(c.Files[c.Selected], res.ScreenX, res.ScreenY)
			}
		}
		return EventConsumed
	}

	if tev, ok := ev.(*tcell.EventKey); ok {
		if tev.Key() == tcell.KeyRune && (tev.Rune() == 'r' || tev.Rune() == 'R') {
			c.Refresh()
			return EventConsumed
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
