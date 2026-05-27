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
	Input        *InputWidget
	Include      *InputWidget
	Exclude      *InputWidget
	ReplaceInput *InputWidget
	Options      SearchOptions
	focusIdx     int
	showFilters  bool
	showReplace  bool
	Groups       []SearchFileGroup
	FlatList     []searchItem
	Selected     int
	ScrollTop    int
	scrollbar    Scrollbar
	WorkDirs     []string
	Searching    bool
	Error        string
	OnOpenMatch  func(path string, line, col int)
	OnReplace    func(filePath string, matches []SearchMatch, replacement string, opts SearchOptions)
	OnReplaceAll func(allMatches map[string][]SearchMatch, replacement string, opts SearchOptions)
	OnPreview    func(filePath string, matches []SearchMatch, replacement string, opts SearchOptions)
}

type searchItem struct {
	IsFile bool
	Group  int
	Match  int
}

func NewSearchWidget() *SearchWidget {
	s := &SearchWidget{}
	s.Input = NewInputWidget(" > ")
	s.Input.Placeholder = "Search"
	s.Include = NewInputWidget(" > ")
	s.Include.Placeholder = "files to include"
	s.Exclude = NewInputWidget(" > ")
	s.Exclude.Placeholder = "files to exclude"
	s.ReplaceInput = NewInputWidget(" > ")
	s.ReplaceInput.Placeholder = "Replace"
	onChange := func(string) { s.runSearch() }
	s.Input.OnChange = onChange
	s.Include.OnChange = onChange
	s.Exclude.OnChange = onChange
	s.Input.Actions = []InputAction{
		{Label: "Aa", OnClick: func() {
			s.Options.CaseSensitive = !s.Options.CaseSensitive
			s.syncOptionActions()
			s.runSearch()
		}},
		{Label: ".*", OnClick: func() {
			s.Options.UseRegex = !s.Options.UseRegex
			s.syncOptionActions()
			s.runSearch()
		}},
	}
	s.ReplaceInput.Actions = []InputAction{
		{Label: "⟳All", OnClick: func() {
			s.ReplaceAllFiles()
		}},
	}
	return s
}

func (s *SearchWidget) syncOptionActions() {
	if len(s.Input.Actions) >= 2 {
		s.Input.Actions[0].Active = s.Options.CaseSensitive
		s.Input.Actions[1].Active = s.Options.UseRegex
	}
}

func (s *SearchWidget) SetReplaceMode(on bool) {
	s.showReplace = on
}

func (s *SearchWidget) IsReplaceMode() bool {
	return s.showReplace
}

func (s *SearchWidget) ToggleReplaceMode() {
	s.showReplace = !s.showReplace
}

func (s *SearchWidget) Refresh() {
	s.runSearch()
}

func (s *SearchWidget) visibleInputs() []*InputWidget {
	inputs := []*InputWidget{s.Input}
	if s.showReplace {
		inputs = append(inputs, s.ReplaceInput)
	}
	if s.showFilters {
		inputs = append(inputs, s.Include, s.Exclude)
	}
	return inputs
}

func (s *SearchWidget) focusedInput() *InputWidget {
	inputs := s.visibleInputs()
	if s.focusIdx >= 0 && s.focusIdx < len(inputs) {
		return inputs[s.focusIdx]
	}
	return s.Input
}

func (s *SearchWidget) SetWorkDirs(dirs []string) {
	s.WorkDirs = dirs
}

func (s *SearchWidget) Focusable() bool { return true }

func (s *SearchWidget) resultsStartY() int {
	base := 3
	if s.showReplace {
		base += 2
	}
	if s.showFilters {
		base += 4
	}
	return base
}

func (s *SearchWidget) CursorPosition() (int, int, bool) {
	r := s.GetRect()
	inp := s.focusedInput()
	y := s.inputY(inp)
	return inp.CursorX(r.X), r.Y + y, true
}

func (s *SearchWidget) inputY(inp *InputWidget) int {
	if inp == s.Input {
		return 0
	}
	if inp == s.ReplaceInput && s.showReplace {
		return 2
	}
	base := 3
	if s.showReplace {
		base += 2
	}
	if inp == s.Include && s.showFilters {
		return base
	}
	if inp == s.Exclude && s.showFilters {
		return base + 2
	}
	return 0
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

	args := []string{"--json", "--max-count=100"}
	if s.Options.CaseSensitive {
		args = append(args, "--case-sensitive")
	} else {
		args = append(args, "--smart-case")
	}
	if !s.Options.UseRegex {
		args = append(args, "--fixed-strings")
	}
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
	y := 1

	if s.showReplace {
		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: '─', Style: term.StyleBorder})
		}
		y++
		s.ReplaceInput.Render(surface, 0, y, w)
		y++
	}

	for x := 0; x < w; x++ {
		surface.SetCell(x, y, term.Cell{Ch: '─', Style: term.StyleBorder})
	}
	y++

	toggleRow := y
	filterToggle := " ▼ "
	if s.showFilters {
		filterToggle = " ◀ "
	}
	ftx := w - len([]rune(filterToggle))
	for i, ch := range filterToggle {
		surface.SetCell(ftx+i, toggleRow, term.Cell{Ch: ch, Style: term.StyleMuted})
	}
	y++

	if s.showFilters {
		s.Include.Render(surface, 0, y, w)
		y++
		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: '─', Style: term.StyleBorder})
		}
		y++

		s.Exclude.Render(surface, 0, y, w)
		y++
		for x := 0; x < w; x++ {
			surface.SetCell(x, y, term.Cell{Ch: '─', Style: term.StyleBorder})
		}
		y++
	}

	startY := y

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
		for x := 0; x < w; x++ {
			surface.SetCell(x, startY, term.Cell{Ch: '─', Style: term.StyleBorder})
		}
		startY++
	}

	visibleH := h - startY
	if visibleH <= 0 {
		return
	}

	if !s.scrollbar.IsDragging() {
		if s.Selected < s.ScrollTop {
			s.ScrollTop = s.Selected
		}
		if s.Selected >= s.ScrollTop+visibleH {
			s.ScrollTop = s.Selected - visibleH + 1
		}
	}

	r := s.GetRect()
	s.scrollbar.X = r.X + w - 1
	s.scrollbar.Y = r.Y + startY
	s.scrollbar.Height = visibleH
	s.scrollbar.TotalItems = len(s.FlatList)
	s.scrollbar.TopItem = s.ScrollTop

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

			if s.showReplace && s.ReplaceInput.Text != "" {
				bx := w - 2
				if bx > x {
					surface.SetCell(bx, y, term.Cell{Ch: '⟳', Style: mutedStyle})
				}
				px := w - 4
				if px > x {
					surface.SetCell(px, y, term.Cell{Ch: '⊙', Style: mutedStyle})
				}
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

	s.scrollbar.Render(surface, w-1, startY)
}

func (s *SearchWidget) toggleRow() int {
	y := 1
	if s.showReplace {
		y += 2
	}
	y++ // separator
	return y
}

func (s *SearchWidget) HandleEvent(ev tcell.Event) EventResult {
	if newTop, consumed := s.scrollbar.HandleEvent(ev); consumed {
		s.ScrollTop = newTop
		return EventConsumed
	}
	switch tev := ev.(type) {
	case *tcell.EventMouse:
		btn := tev.Buttons()
		if btn&tcell.Button1 != 0 {
			mx, my := tev.Position()
			r := s.GetRect()
			localX := mx - r.X
			localY := my - r.Y

			if localY == 0 {
				if s.Input.HandleMouseClick(localX, localY) {
					return EventConsumed
				}
				s.focusIdx = 0
				return EventConsumed
			}

			if s.showReplace && localY == 2 {
				if s.ReplaceInput.HandleMouseClick(localX, localY) {
					return EventConsumed
				}
				inputs := s.visibleInputs()
				for i, inp := range inputs {
					if inp == s.ReplaceInput {
						s.focusIdx = i
						break
					}
				}
				return EventConsumed
			}

			tRow := s.toggleRow()
			if localY == tRow {
				if localX >= r.W-3 {
					s.showFilters = !s.showFilters
					inputs := s.visibleInputs()
					if s.focusIdx >= len(inputs) {
						s.focusIdx = 0
					}
					return EventConsumed
				}
			}

			if s.showFilters {
				inputs := s.visibleInputs()
				for i, inp := range inputs {
					iy := s.inputY(inp)
					if localY == iy {
						s.focusIdx = i
						return EventConsumed
					}
				}
			}

			startY := s.resultsStartY()
			if len(s.Groups) > 0 {
				startY++
			}

			idx := s.ScrollTop + (localY - startY)
			if idx >= 0 && idx < len(s.FlatList) {
				item := s.FlatList[idx]
				if s.showReplace && s.ReplaceInput.Text != "" && item.IsFile {
					if localX >= r.W-3 && localX <= r.W-2 {
						s.replaceInFile(item.Group)
						return EventConsumed
					}
					if localX >= r.W-5 && localX <= r.W-4 {
						s.previewFile(item.Group)
						return EventConsumed
					}
				}
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
		if tev.Modifiers()&tcell.ModAlt != 0 && tev.Key() == tcell.KeyRune {
			switch tev.Rune() {
			case 'c':
				s.Options.CaseSensitive = !s.Options.CaseSensitive
				s.syncOptionActions()
				s.runSearch()
				return EventConsumed
			case 'r':
				s.Options.UseRegex = !s.Options.UseRegex
				s.syncOptionActions()
				s.runSearch()
				return EventConsumed
			}
		}
		switch tev.Key() {
		case tcell.KeyTab:
			inputs := s.visibleInputs()
			if len(inputs) > 1 {
				s.focusIdx = (s.focusIdx + 1) % len(inputs)
			}
			return EventConsumed
		case tcell.KeyBacktab:
			inputs := s.visibleInputs()
			if len(inputs) > 1 {
				s.focusIdx = (s.focusIdx + len(inputs) - 1) % len(inputs)
			}
			return EventConsumed
		case tcell.KeyEnter:
			if s.focusIdx == 0 && len(s.FlatList) == 0 {
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

func (s *SearchWidget) replaceInFile(groupIdx int) {
	if groupIdx < 0 || groupIdx >= len(s.Groups) {
		return
	}
	g := s.Groups[groupIdx]
	if s.OnReplace != nil {
		s.OnReplace(g.FilePath, g.Matches, s.ReplaceInput.Text, s.Options)
	}
}

func (s *SearchWidget) previewFile(groupIdx int) {
	if groupIdx < 0 || groupIdx >= len(s.Groups) {
		return
	}
	g := s.Groups[groupIdx]
	if s.OnPreview != nil {
		s.OnPreview(g.FilePath, g.Matches, s.ReplaceInput.Text, s.Options)
	}
}

func (s *SearchWidget) ReplaceAllFiles() {
	if s.OnReplaceAll == nil {
		return
	}
	allMatches := make(map[string][]SearchMatch)
	for _, g := range s.Groups {
		allMatches[g.FilePath] = g.Matches
	}
	s.OnReplaceAll(allMatches, s.ReplaceInput.Text, s.Options)
}

func (s *SearchWidget) activateSelected() {
	if s.Selected < 0 || s.Selected >= len(s.FlatList) {
		return
	}
	item := s.FlatList[s.Selected]
	if item.IsFile {
		if s.showReplace && s.ReplaceInput.Text != "" {
			s.previewFile(item.Group)
		} else {
			g := &s.Groups[item.Group]
			g.Expanded = !g.Expanded
			s.flatten()
		}
	} else {
		if s.showReplace && s.ReplaceInput.Text != "" {
			s.previewFile(item.Group)
		} else {
			m := s.Groups[item.Group].Matches[item.Match]
			if s.OnOpenMatch != nil {
				s.OnOpenMatch(m.FilePath, m.LineNum, m.ColStart)
			}
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
