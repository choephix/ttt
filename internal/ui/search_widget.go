package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"ttt/internal/term"

	"github.com/gdamore/tcell/v2"
)

type SearchMatch struct {
	FilePath string
	LineNum  int
	ColStart int
	ColEnd   int
	LineText string
}

type SearchFileGroup struct {
	FilePath string
	RelPath  string
	Matches  []SearchMatch
	Expanded bool
}

type SearchWidget struct {
	BaseWidget
	Input     *InputWidget
	Groups    []SearchFileGroup
	FlatList  []searchItem
	Selected  int
	ScrollTop int
	WorkDir   string
	Searching bool
	Error     string
	OnOpenMatch func(path string, line, col int)
}

type searchItem struct {
	IsFile bool
	Group  int
	Match  int
}

func NewSearchWidget() *SearchWidget {
	s := &SearchWidget{}
	s.Input = NewInputWidget(" > ")
	s.Input.OnChange = func(text string) {
		s.runSearch()
	}
	return s
}

func (s *SearchWidget) SetWorkDir(dir string) {
	s.WorkDir = dir
}

func (s *SearchWidget) Focusable() bool { return true }

func (s *SearchWidget) CursorPosition() (int, int, bool) {
	r := s.GetRect()
	return s.Input.CursorX(r.X), r.Y, true
}

func (s *SearchWidget) runSearch() {
	s.Groups = nil
	s.FlatList = nil
	s.Selected = 0
	s.ScrollTop = 0
	s.Error = ""

	if s.Input.Text == "" {
		return
	}

	if _, err := exec.LookPath("rg"); err != nil {
		s.Error = "ripgrep (rg) not found"
		return
	}

	s.Searching = true
	dir := s.WorkDir
	if dir == "" {
		dir = "."
	}

	cmd := exec.Command("rg", "--json", "--smart-case", "--max-count=100", s.Input.Text, dir)
	out, err := cmd.Output()
	s.Searching = false

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return
			}
		}
		s.Error = "search failed"
		return
	}

	groupMap := map[string]int{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		var msg rgMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Type != "match" {
			continue
		}
		filePath := msg.Data.Path.Text
		absPath := filePath
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(dir, absPath)
		}
		relPath := filePath
		if dir != "" {
			if r, err := filepath.Rel(dir, absPath); err == nil {
				relPath = r
			}
		}

		lineText := msg.Data.Lines.Text
		lineText = strings.TrimRight(lineText, "\n\r")

		for _, sub := range msg.Data.Submatches {
			match := SearchMatch{
				FilePath: absPath,
				LineNum:  msg.Data.LineNumber,
				ColStart: sub.Start,
				ColEnd:   sub.End,
				LineText: lineText,
			}

			idx, ok := groupMap[absPath]
			if !ok {
				idx = len(s.Groups)
				groupMap[absPath] = idx
				s.Groups = append(s.Groups, SearchFileGroup{
					FilePath: absPath,
					RelPath:  relPath,
					Expanded: true,
				})
			}
			s.Groups[idx].Matches = append(s.Groups[idx].Matches, match)
		}
	}

	s.flatten()
}

type rgMessage struct {
	Type string         `json:"type"`
	Data rgMatchData    `json:"data"`
}

type rgMatchData struct {
	Path       rgText       `json:"path"`
	Lines      rgText       `json:"lines"`
	LineNumber int          `json:"line_number"`
	Submatches []rgSubmatch `json:"submatches"`
}

type rgText struct {
	Text string `json:"text"`
}

type rgSubmatch struct {
	Match rgText `json:"match"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (s *SearchWidget) flatten() {
	s.FlatList = nil
	for gi, g := range s.Groups {
		s.FlatList = append(s.FlatList, searchItem{IsFile: true, Group: gi})
		if g.Expanded {
			for mi := range g.Matches {
				s.FlatList = append(s.FlatList, searchItem{IsFile: false, Group: gi, Match: mi})
			}
		}
	}
}

func (s *SearchWidget) Render(surface *RenderSurface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' '})

	if h < 2 {
		return
	}

	s.Input.Render(surface, 0, 0, w)

	for x := 0; x < w; x++ {
		surface.SetCell(x, 1, term.Cell{Ch: '─', Style: term.StyleBorder})
	}

	startY := 2

	if s.Error != "" {
		for i, ch := range s.Error {
			if i < w {
				surface.SetCell(i, startY, term.Cell{Ch: ch, Style: term.StyleMuted})
			}
		}
		return
	}

	if s.Input.Text != "" && len(s.FlatList) == 0 && !s.Searching {
		msg := "No results"
		for i, ch := range msg {
			if i < w {
				surface.SetCell(i, startY, term.Cell{Ch: ch, Style: term.StyleMuted})
			}
		}
		return
	}

	if len(s.Groups) > 0 {
		totalMatches := 0
		for _, g := range s.Groups {
			totalMatches += len(g.Matches)
		}
		summary := fmt.Sprintf("%d results in %d files", totalMatches, len(s.Groups))
		for i, ch := range summary {
			if i < w {
				surface.SetCell(i, startY, term.Cell{Ch: ch, Style: term.StyleMuted})
			}
		}
		startY++
	}

	visibleH := h - startY
	if visibleH <= 0 {
		return
	}

	if s.Selected < s.ScrollTop {
		s.ScrollTop = s.Selected
	}
	if s.Selected >= s.ScrollTop+visibleH {
		s.ScrollTop = s.Selected - visibleH + 1
	}

	for i := 0; i < visibleH; i++ {
		idx := s.ScrollTop + i
		if idx >= len(s.FlatList) {
			break
		}
		item := s.FlatList[idx]
		y := startY + i

		style := term.StyleDefault
		if idx == s.Selected {
			style = term.StyleSidebarSelected
		}

		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: style})
		}

		if item.IsFile {
			g := s.Groups[item.Group]
			chevron := '▼'
			if !g.Expanded {
				chevron = '▶'
			}
			x := 0
			surface.SetCell(x, y, term.Cell{Ch: chevron, Style: style})
			x += 2

			for _, ch := range g.RelPath {
				if x >= w {
					break
				}
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: style})
				x++
			}

			countStr := fmt.Sprintf(" (%d)", len(g.Matches))
			mutedStyle := term.StyleMuted
			if idx == s.Selected {
				mutedStyle = style
			}
			for _, ch := range countStr {
				if x >= w {
					break
				}
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: mutedStyle})
				x++
			}
		} else {
			m := s.Groups[item.Group].Matches[item.Match]
			x := 2

			lineStr := fmt.Sprintf("%d: ", m.LineNum)
			mutedStyle := term.StyleMuted
			if idx == s.Selected {
				mutedStyle = style
			}
			for _, ch := range lineStr {
				if x >= w {
					break
				}
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: mutedStyle})
				x++
			}

			trimmed := strings.TrimLeft(m.LineText, " \t")
			trimOff := len(m.LineText) - len(trimmed)
			for ci, ch := range trimmed {
				if x >= w {
					break
				}
				cs := style
				origCol := ci + trimOff
				if origCol >= m.ColStart && origCol < m.ColEnd {
					cs = term.StyleSearchMatch
				}
				surface.SetCell(x, y, term.Cell{Ch: ch, Style: cs})
				x++
			}
		}
	}
}

func (s *SearchWidget) HandleEvent(ev tcell.Event) EventResult {
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		btn := tev.Buttons()
		if btn&tcell.Button1 != 0 {
			_, my := tev.Position()
			r := s.GetRect()
			localY := my - r.Y

			startY := 3
			if len(s.Groups) > 0 {
				startY = 3
			}

			idx := s.ScrollTop + (localY - startY)
			if idx >= 0 && idx < len(s.FlatList) {
				s.Selected = idx
				s.activateSelected()
			}
			return EventConsumed
		}
		if btn&tcell.WheelUp != 0 {
			s.ScrollTop -= 3
			if s.ScrollTop < 0 {
				s.ScrollTop = 0
			}
			return EventConsumed
		}
		if btn&tcell.WheelDown != 0 {
			max := len(s.FlatList) - 5
			if max < 0 {
				max = 0
			}
			s.ScrollTop += 3
			if s.ScrollTop > max {
				s.ScrollTop = max
			}
			return EventConsumed
		}
	case *tcell.EventKey:
		switch tev.Key() {
		case tcell.KeyEnter:
			if len(s.FlatList) == 0 {
				s.runSearch()
			} else {
				s.activateSelected()
			}
			return EventConsumed
		case tcell.KeyUp:
			if s.Selected > 0 {
				s.Selected--
			}
			return EventConsumed
		case tcell.KeyDown:
			if s.Selected < len(s.FlatList)-1 {
				s.Selected++
			}
			return EventConsumed
		case tcell.KeyLeft:
			s.collapseSelected()
			return EventConsumed
		case tcell.KeyRight:
			s.expandSelected()
			return EventConsumed
		default:
			if s.Input.HandleEvent(ev) == EventConsumed {
				return EventConsumed
			}
		}
	}

	return EventIgnored
}

func (s *SearchWidget) activateSelected() {
	if s.Selected < 0 || s.Selected >= len(s.FlatList) {
		return
	}
	item := s.FlatList[s.Selected]
	if item.IsFile {
		g := &s.Groups[item.Group]
		g.Expanded = !g.Expanded
		s.flatten()
	} else {
		m := s.Groups[item.Group].Matches[item.Match]
		if s.OnOpenMatch != nil {
			s.OnOpenMatch(m.FilePath, m.LineNum, m.ColStart)
		}
	}
}

func (s *SearchWidget) collapseSelected() {
	if s.Selected < 0 || s.Selected >= len(s.FlatList) {
		return
	}
	item := s.FlatList[s.Selected]
	if item.IsFile {
		s.Groups[item.Group].Expanded = false
		s.flatten()
	}
}

func (s *SearchWidget) expandSelected() {
	if s.Selected < 0 || s.Selected >= len(s.FlatList) {
		return
	}
	item := s.FlatList[s.Selected]
	if item.IsFile {
		s.Groups[item.Group].Expanded = true
		s.flatten()
	}
}
