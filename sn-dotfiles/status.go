package sndotfiles

import (
	"fmt"

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
	var remote tagsWithNotes

	remote, err = get(session, debug)
	if err != nil {
		return diffs, msg, err
	}

	return status(remote, home, paths, debug)
}

func status(twn tagsWithNotes, home string, paths []string, debug bool) (diffs []ItemDiff, msg string, err error) {
	debugPrint(debug, fmt.Sprintf("status | %d remote items", len(twn)))

	err = checkNoteTagConflicts(twn)
	if err != nil {
		return
	}

	if len(twn) == 0 {
		msg = "no dotfiles being tracked"
		return
	}

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
