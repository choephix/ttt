package main

import (
	"fmt"

	"github.com/eugenioenko/ttt/internal/github"

	"github.com/gdamore/tcell/v2"
)

type prFetchResult struct {
	url   string
	info  *github.PRInfo
	diffs map[string]string
	err   error
}

func (a *App) fetchAndOpenPR(url string) {
	owner, repo, number, err := github.ParsePRURL(url)
	if err != nil {
		a.StatusError("Invalid PR URL: " + err.Error())
		return
	}

	a.changes.Loading = true
	a.StatusNotify(fmt.Sprintf("Fetching PR #%d...", number))

	go func() {
		info, err := github.FetchPRInfo(owner, repo, number)
		if err != nil {
			a.screen.PostEvent(tcell.NewEventInterrupt(&prFetchResult{url: url, err: err}))
			return
		}

		diffText, err := github.FetchPRDiff(owner, repo, number)
		if err != nil {
			a.screen.PostEvent(tcell.NewEventInterrupt(&prFetchResult{url: url, err: err}))
			return
		}

		diffs := github.SplitMultiFileDiff(diffText)
		a.screen.PostEvent(tcell.NewEventInterrupt(&prFetchResult{url: url, info: info, diffs: diffs}))
	}()
}
