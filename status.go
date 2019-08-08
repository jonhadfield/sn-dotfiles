package sndotfiles

import (
	"errors"
	"fmt"

	"github.com/ryanuber/columnize"

	"github.com/fatih/color"
	"github.com/jonhadfield/gosn"
)

func colourDiff(diff string) string {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	switch diff {
	case identical:
		return green(diff)
	case localMissing:
		return red(diff)
	case localNewer:
		return yellow(diff)
	case untracked:
		return yellow(diff)
	case remoteNewer:
		return yellow(diff)
	}
	return diff
}

func Status(session gosn.Session, home string, paths []string, quiet, debug bool) (diffs []ItemDiff, err error) {
	remote, err := get(session)
	if err != nil {
		return diffs, err
	}
	debugPrint(debug, fmt.Sprintf("status | %d remote items", len(remote)))
	err = preflight(remote, paths)
	if err != nil {
		return
	}
	if len(remote) == 0 {
		return diffs, errors.New("no dotfiles being tracked")
	}
	bold := color.New(color.Bold).SprintFunc()

	diffs, err = diff(remote, home, paths, debug)
	if err != nil {
		return diffs, err
	}
	debugPrint(debug, fmt.Sprintf("status | %d diffs generated", len(diffs)))
	var lines []string
	if len(diffs) == 0 {
		return diffs, err
	}
	for _, diff := range diffs {
		lines = append(lines, fmt.Sprintf("%s | %s \n", bold(diff.homeRelPath), colourDiff(diff.diff)))
	}
	if !quiet {
		fmt.Println(columnize.SimpleFormat(lines))
	}
	return diffs, err
}
