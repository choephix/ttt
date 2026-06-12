package app

import (
	"fmt"

	"github.com/eugenioenko/ttt/internal/github"

	"github.com/gdamore/tcell/v2"
)

type PrFetchResult struct {
	URL   string
	Info  *github.PRInfo
	Diffs map[string]string
	Err   error
}

type DiffContentResult struct {
	TabName  string
	OldLines []string
	NewLines []string
	Err      error
}

func (a *App) FetchAndOpenPR(url string) {
	owner, repo, number, err := github.ParsePRURL(url)
	if err != nil {
		a.StatusError("Invalid PR URL: " + err.Error())
		return
	}

	a.Changes.Loading = true
	a.StatusNotify(fmt.Sprintf("Fetching PR #%d...", number))

	go func() {
		info, err := github.FetchPRInfo(owner, repo, number)
		if err != nil {
			a.Screen.PostEvent(tcell.NewEventInterrupt(&PrFetchResult{URL: url, Err: err}))
			return
		}

		diffText, err := github.FetchPRDiff(owner, repo, number)
		if err != nil {
			a.Screen.PostEvent(tcell.NewEventInterrupt(&PrFetchResult{URL: url, Err: err}))
			return
		}

		diffs := github.SplitMultiFileDiff(diffText)
		a.Screen.PostEvent(tcell.NewEventInterrupt(&PrFetchResult{URL: url, Info: info, Diffs: diffs}))
	}()
}
