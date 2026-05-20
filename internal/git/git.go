package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

type FileStatus struct {
	Status string
	Path   string
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
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		files = append(files, FileStatus{Status: status, Path: path})
	}
	return files, nil
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
