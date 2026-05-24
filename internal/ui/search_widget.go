package ui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"github.com/eugenioenko/ttt/internal/term"

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
	Input       *InputWidget
	Include     *InputWidget
	Exclude     *InputWidget
	focusIdx    int
	showFilters bool
	Groups      []SearchFileGroup
	FlatList    []searchItem
	Selected    int
	ScrollTop   int
	WorkDirs    []string
	Searching   bool
	Error       string
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
	s.Include = NewInputWidget(" > ")
	s.Exclude = NewInputWidget(" > ")
	onChange := func(string) { s.runSearch() }
	s.Input.OnChange = onChange
	s.Include.OnChange = onChange
	s.Exclude.OnChange = onChange
	return s
}

func (s *SearchWidget) focusedInput() *InputWidget {
	switch s.focusIdx {
	case 1:
		return s.Include
	case 2:
		return s.Exclude
	default:
		return s.Input
	}
}

func (s *SearchWidget) SetWorkDirs(dirs []string) {
	s.WorkDirs = dirs
}

func (s *SearchWidget) Focusable() bool { return true }

func (s *SearchWidget) resultsStartY() int {
	if s.showFilters {
		return 9
	}
	return 3
}

func (s *SearchWidget) CursorPosition() (int, int, bool) {
	r := s.GetRect()
	if !s.showFilters || s.focusIdx == 0 {
		return s.Input.CursorX(r.X), r.Y, true
	}
	if s.focusIdx == 1 {
		return s.Include.CursorX(r.X), r.Y + 4, true
	}
	return s.Exclude.CursorX(r.X), r.Y + 7, true
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
	dirs := s.WorkDirs
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	args := []string{"--json", "--smart-case", "--max-count=100"}
	for _, g := range strings.Split(s.Include.Text, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			args = append(args, "--glob", g)
		}
	}
	for _, g := range strings.Split(s.Exclude.Text, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			args = append(args, "--glob", "!"+g)
		}
	}
	args = append(args, s.Input.Text)
	args = append(args, dirs...)
	cmd := exec.Command("rg", args...)
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
		if !filepath.IsAbs(absPath) && len(dirs) > 0 {
			absPath = filepath.Join(dirs[0], absPath)
		}
		relPath := filePath
		for _, d := range dirs {
			if r, err := filepath.Rel(d, absPath); err == nil && !strings.HasPrefix(r, "..") {
				relPath = r
				break
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

	toggle := " ▼ "
	if s.showFilters {
		toggle = " ◀ "
	}
	tx := w - len([]rune(toggle))
	for i, ch := range toggle {
		surface.SetCell(tx+i, 2, term.Cell{Ch: ch, Style: term.StyleMuted})
	}

	startY := s.resultsStartY()

	if s.showFilters {
		includeLabel := "files to include"
		for i, ch := range includeLabel {
			if i+1 < w {
				surface.SetCell(i+1, 3, term.Cell{Ch: ch, Style: term.StyleMuted})
			}
		}
		s.Include.Render(surface, 0, 4, w)
		for x := 0; x < w; x++ {
			surface.SetCell(x, 5, term.Cell{Ch: '─', Style: term.StyleBorder})
		}

		excludeLabel := "files to exclude"
		for i, ch := range excludeLabel {
			if i+1 < w {
				surface.SetCell(i+1, 6, term.Cell{Ch: ch, Style: term.StyleMuted})
			}
		}
		s.Exclude.Render(surface, 0, 7, w)
		for x := 0; x < w; x++ {
			surface.SetCell(x, 8, term.Cell{Ch: '─', Style: term.StyleBorder})
		}
	}

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
			mx, my := tev.Position()
			r := s.GetRect()
			localY := my - r.Y

			if localY == 0 {
				s.focusIdx = 0
				return EventConsumed
			}

			if localY == 2 && mx >= r.X+r.W-3 {
				s.showFilters = !s.showFilters
				if !s.showFilters && s.focusIdx > 0 {
					s.focusIdx = 0
				}
				return EventConsumed
			}

			if s.showFilters {
				if localY == 4 {
					s.focusIdx = 1
					return EventConsumed
				}
				if localY == 7 {
					s.focusIdx = 2
					return EventConsumed
				}
			}

			startY := s.resultsStartY()
			if len(s.Groups) > 0 {
				startY++
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
		case tcell.KeyTab:
			if s.showFilters {
				s.focusIdx = (s.focusIdx + 1) % 3
			}
			return EventConsumed
		case tcell.KeyBacktab:
			if s.showFilters {
				s.focusIdx = (s.focusIdx + 2) % 3
			}
			return EventConsumed
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
			if s.focusedInput().HandleEvent(ev) == EventConsumed {
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
