package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FileStatus struct {
	Status string
	Path   string
	Staged bool
}

func RepoRoot(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func IsRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func StatusFiles(dir string) ([]FileStatus, error) {
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain", "-u")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []FileStatus
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		x := line[0] // index (staged) status
		y := line[1] // worktree (unstaged) status
		path := strings.TrimSpace(line[3:])

		if x != ' ' && x != '?' {
			files = append(files, FileStatus{Status: string(x), Path: path, Staged: true})
		}
		if y != ' ' {
			st := string(y)
			if x == '?' && y == '?' {
				st = "?"
			}
			files = append(files, FileStatus{Status: st, Path: path, Staged: false})
		}
	}
	return files, nil
}

func Stage(dir, path string) error {
	cmd := exec.Command("git", "-C", dir, "add", "--", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Unstage(dir, path string) error {
	cmd := exec.Command("git", "-C", dir, "reset", "HEAD", "--", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Commit(dir, message string) error {
	cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Pull(dir string) error {
	cmd := exec.Command("git", "-C", dir, "pull")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func Push(dir string) error {
	cmd := exec.Command("git", "-C", dir, "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func BranchName(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type BlameInfo struct {
	Author string
	Time   time.Time
	Summary string
}

func BlameLine(dir, file string, line int) *BlameInfo {
	lineStr := fmt.Sprintf("%d,%d", line, line)
	cmd := exec.Command("git", "-C", dir, "blame", "-L", lineStr,
		"--porcelain", "--", file)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	info := &BlameInfo{}
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "author ") {
			info.Author = strings.TrimPrefix(l, "author ")
		} else if strings.HasPrefix(l, "author-time ") {
			ts, err := strconv.ParseInt(strings.TrimPrefix(l, "author-time "), 10, 64)
			if err == nil {
				info.Time = time.Unix(ts, 0)
			}
		} else if strings.HasPrefix(l, "summary ") {
			info.Summary = strings.TrimPrefix(l, "summary ")
		}
	}

	if info.Author == "" && info.Summary == "" {
		return nil
	}
	// Uncommitted changes
	if strings.HasPrefix(info.Author, "Not Committed") {
		return nil
	}
	return info
}

func FormatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months <= 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

func IgnoredFiles(dir string, paths []string) map[string]bool {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"-C", dir, "check-ignore"}, paths...)
	cmd := exec.Command("git", args...)
	out, _ := cmd.Output()
	result := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result[line] = true
		}
	}
	return result
}

func DiffFile(dir, path string) (string, error) {
	absPath := filepath.Join(dir, path)
	cmd := exec.Command("git", "-C", dir, "diff", "--", absPath)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return string(out), nil
		}
		return "", err
	}
	if len(out) == 0 {
		cmd = exec.Command("git", "-C", dir, "diff", "--cached", "--", absPath)
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}
	if len(out) == 0 {
		cmd = exec.Command("git", "-C", dir, "diff", "HEAD", "--", absPath)
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}
	return string(out), nil
}
