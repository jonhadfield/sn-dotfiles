package sndotfiles

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/jonhadfield/gosn"
	"github.com/ryanuber/columnize"
)

// Status compares and then outputs status of all items (or a subset defined by Paths param):
// - local items that missing
// - local items that are newer
// - remote items that are newer
// - local items that are untracked (if Paths specified)
// - identical local and remote items
func Status(session gosn.Session, home string, paths []string, debug bool) (diffs []ItemDiff, msg string, err error) {
	remote, err := get(session)
	if err != nil {
		return diffs, msg, err
	}
	return status(remote, home, paths, debug)
}

func status(twn tagsWithNotes, home string, paths []string, debug bool) (diffs []ItemDiff, msg string, err error) {
	debugPrint(debug, fmt.Sprintf("status | %d remote items", len(twn)))
	err = preflight(twn, paths)
	if err != nil {
		return
	}
	if len(twn) == 0 {
		msg = "no dotfiles being tracked"
		return
	}
	bold := color.New(color.Bold).SprintFunc()

	diffs, err = compare(twn, home, paths, []string{}, debug)
	if err != nil {
		return diffs, msg, err
	}
	debugPrint(debug, fmt.Sprintf("status | %d diffs generated", len(diffs)))
	if len(diffs) == 0 {
		return diffs, msg, err
	}
	lines := make([]string, len(diffs))
	for i, diff := range diffs {
		lines[i] = fmt.Sprintf("%s | %s \n", bold(diff.homeRelPath), colourDiff(diff.diff))
	}
	msg = columnize.SimpleFormat(lines)
	return diffs, msg, err
}

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
	default:
		return diff
	}
}
