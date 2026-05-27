package diff

import "strings"

type LineKind int

const (
	Blank   LineKind = iota
	Context
	Added
	Deleted
)

type SideLine struct {
	Num  int
	Text string
	Kind LineKind
}

type DiffLine struct {
	Left  SideLine
	Right SideLine
}

type Hunk struct {
	Header string
	Lines  []DiffLine
}

type FileDiff struct {
	OldName string
	NewName string
	Hunks   []Hunk
}

func (f *FileDiff) AllLines() []DiffLine {
	var lines []DiffLine
	for i, h := range f.Hunks {
		if i > 0 {
			lines = append(lines, DiffLine{
				Left:  SideLine{Kind: Blank, Text: h.Header},
				Right: SideLine{Kind: Blank, Text: h.Header},
			})
		}
		lines = append(lines, h.Lines...)
	}
	return lines
}

func Parse(unified string) FileDiff {
	var fd FileDiff
	lines := strings.Split(unified, "\n")

	var curHunk *Hunk
	var delBuf []string
	var addBuf []string
	oldNum, newNum := 0, 0

	flush := func() {
		if curHunk == nil {
			delBuf = nil
			addBuf = nil
			return
		}
		maxLen := len(delBuf)
		if len(addBuf) > maxLen {
			maxLen = len(addBuf)
		}
		for i := 0; i < maxLen; i++ {
			dl := DiffLine{}
			if i < len(delBuf) {
				dl.Left = SideLine{Num: oldNum, Text: delBuf[i], Kind: Deleted}
				oldNum++
			} else {
				dl.Left = SideLine{Kind: Blank}
			}
			if i < len(addBuf) {
				dl.Right = SideLine{Num: newNum, Text: addBuf[i], Kind: Added}
				newNum++
			} else {
				dl.Right = SideLine{Kind: Blank}
			}
			curHunk.Lines = append(curHunk.Lines, dl)
		}
		delBuf = nil
		addBuf = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "--- ") {
			name := line[4:]
			if strings.HasPrefix(name, "a/") {
				name = name[2:]
			}
			fd.OldName = name
			continue
		}
		if strings.HasPrefix(line, "+++ ") {
			name := line[4:]
			if strings.HasPrefix(name, "b/") {
				name = name[2:]
			}
			fd.NewName = name
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			flush()
			if curHunk != nil {
				fd.Hunks = append(fd.Hunks, *curHunk)
			}
			curHunk = &Hunk{Header: line}
			oldNum, newNum = parseHunkHeader(line)
			continue
		}
		if strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "\\ ") {
			continue
		}
		if curHunk == nil {
			continue
		}
		if strings.HasPrefix(line, "-") {
			delBuf = append(delBuf, line[1:])
		} else if strings.HasPrefix(line, "+") {
			addBuf = append(addBuf, line[1:])
		} else {
			flush()
			text := line
			if len(text) > 0 && text[0] == ' ' {
				text = text[1:]
			}
			curHunk.Lines = append(curHunk.Lines, DiffLine{
				Left:  SideLine{Num: oldNum, Text: text, Kind: Context},
				Right: SideLine{Num: newNum, Text: text, Kind: Context},
			})
			oldNum++
			newNum++
		}
	}

	flush()
	if curHunk != nil {
		fd.Hunks = append(fd.Hunks, *curHunk)
	}

	return fd
}

func parseHunkHeader(header string) (oldStart, newStart int) {
	// @@ -oldStart,oldCount +newStart,newCount @@
	parts := strings.Split(header, " ")
	for _, p := range parts {
		if strings.HasPrefix(p, "-") && strings.Contains(p, ",") {
			n := 0
			for _, ch := range p[1:] {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				} else {
					break
				}
			}
			oldStart = n
		}
		if strings.HasPrefix(p, "+") && strings.Contains(p, ",") {
			n := 0
			for _, ch := range p[1:] {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				} else {
					break
				}
			}
			newStart = n
		}
	}
	return oldStart, newStart
}
