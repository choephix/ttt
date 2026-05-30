package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type PRFile struct {
	Path   string
	Status string // A, M, D, R
}

type PRInfo struct {
	Owner  string
	Repo   string
	Number int
	Title  string
	Files  []PRFile
}

func IsGHInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func ParsePRURL(url string) (owner, repo string, number int, err error) {
	url = strings.TrimRight(url, "/")
	url = strings.SplitN(url, "?", 2)[0]
	url = strings.SplitN(url, "#", 2)[0]
	parts := strings.Split(url, "/")
	for i, p := range parts {
		if p == "pull" && i >= 2 && i+1 < len(parts) {
			owner = parts[i-2]
			repo = parts[i-1]
			n, e := strconv.Atoi(parts[i+1])
			if e != nil {
				return "", "", 0, fmt.Errorf("invalid PR number: %s", parts[i+1])
			}
			return owner, repo, n, nil
		}
	}
	return "", "", 0, fmt.Errorf("could not parse PR URL: %s", url)
}

func FetchPRInfo(owner, repo string, number int) (*PRInfo, error) {
	repoArg := owner + "/" + repo
	cmd := exec.Command("gh", "pr", "view", strconv.Itoa(number),
		"--repo", repoArg,
		"--json", "title,number,files")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr view failed: %w", err)
	}
	var result struct {
		Title  string `json:"title"`
		Number int    `json:"number"`
		Files  []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"files"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse gh output: %w", err)
	}
	info := &PRInfo{
		Owner:  owner,
		Repo:   repo,
		Number: result.Number,
		Title:  result.Title,
	}
	for _, f := range result.Files {
		status := "M"
		switch f.Status {
		case "added":
			status = "A"
		case "deleted":
			status = "D"
		case "renamed":
			status = "R"
		case "copied":
			status = "A"
		}
		info.Files = append(info.Files, PRFile{Path: f.Path, Status: status})
	}
	return info, nil
}

func FetchPRDiff(owner, repo string, number int) (string, error) {
	repoArg := owner + "/" + repo
	cmd := exec.Command("gh", "pr", "diff", strconv.Itoa(number), "--repo", repoArg)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh pr diff failed: %w", err)
	}
	return string(out), nil
}

func SplitMultiFileDiff(unified string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(unified, "\n")
	var currentFile string
	var currentLines []string

	flush := func() {
		if currentFile != "" && len(currentLines) > 0 {
			result[currentFile] = strings.Join(currentLines, "\n")
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				bPath := parts[len(parts)-1]
				if strings.HasPrefix(bPath, "b/") {
					currentFile = bPath[2:]
				} else {
					currentFile = bPath
				}
			}
			currentLines = []string{line}
		} else {
			currentLines = append(currentLines, line)
		}
	}
	flush()
	return result
}
