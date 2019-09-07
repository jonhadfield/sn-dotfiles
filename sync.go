package sndotfiles

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jonhadfield/gosn"
	"github.com/ryanuber/columnize"
)

// Sync compares local and remote items and then:
// - pulls remotes if locals are older or missing
// - pushes locals if remotes are newer
func Sync(session gosn.Session, home string, paths, exclude []string, debug bool) (noPushed, noPulled int, msg string, err error) {
	var remote tagsWithNotes
	remote, err = get(session)
	if err != nil {
		return
	}
	err = preflight(remote, paths)
	if err != nil {
		return
	}
	return sync(session, remote, home, exclude, debug)
}

func ensureTrailingPathSep(in string) string {
	if strings.HasSuffix(in, string(os.PathSeparator)) {
		return in
	}
	return in + string(os.PathSeparator)
}

func matchesPathsToExclude(home, path string, pathsToExclude []string) bool {
	fmt.Println("in matchesPathsToExclude")
	for _, pte := range pathsToExclude {
		homeStrippedPath := stripHome(pte, home)
		// return match if paths match exactly
		if homeStrippedPath == path {
			return true
		}
		// return match if pte is a parent of the path
		if strings.HasPrefix(path, ensureTrailingPathSep(homeStrippedPath)) {
			return true
		}
	}
	return false
}

func sync(session gosn.Session, twn tagsWithNotes, home string, exclude []string, debug bool) (noPushed, noPulled int, msg string, err error) {
	var itemDiffs []ItemDiff
	itemDiffs, err = diff(twn, home, nil, exclude, debug)
	if err != nil {
		if strings.Contains(err.Error(), "tags with notes not supplied") {
			err = errors.New("no remote dotfiles found")
		}
		return
	}

	var itemsToPush, itemsToPull []ItemDiff
	var itemsToSync bool
	for _, itemDiff := range itemDiffs {
		// check if itemDiff is for a path to be excluded
		if matchesPathsToExclude(home, itemDiff.homeRelPath, exclude) {
			debugPrint(debug, fmt.Sprintf("sync | excluding: %s", itemDiff.homeRelPath))
			continue
		}

		switch itemDiff.diff {
		case localNewer:
			//push
			debugPrint(debug, fmt.Sprintf("sync | local %s is newer", itemDiff.homeRelPath))
			itemDiff.remote.Content.SetText(itemDiff.local)
			itemsToPush = append(itemsToPush, itemDiff)
			itemsToSync = true
		case localMissing:
			// pull
			debugPrint(debug, fmt.Sprintf("sync | %s is missing", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		case remoteNewer:
			// pull
			debugPrint(debug, fmt.Sprintf("sync | remote %s is newer", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		}
	}
	bold := color.New(color.Bold).SprintFunc()

	// check items to sync
	if !itemsToSync {
		msg = fmt.Sprint(bold("nothing to do"))
		return
	}

	// push
	if len(itemsToPush) > 0 {
		_, err = push(session, itemsToPush)
		noPushed = len(itemsToPush)
		if err != nil {
			return
		}
	}

	res := make([]string, len(itemsToPush))
	green := color.New(color.FgGreen).SprintFunc()
	strPushed := green("pushed")
	strPulled := green("pulled")

	for _, pushItem := range itemsToPush {
		line := fmt.Sprintf("%s | %s", bold(addDot(pushItem.homeRelPath)), strPushed)
		res = append(res, line)
	}

	// pull
	if err = pull(itemsToPull); err != nil {
		return
	}
	noPulled = len(itemsToPull)

	for _, pullItem := range itemsToPull {
		line := fmt.Sprintf("%s | %s\n", bold(addDot(pullItem.homeRelPath)), strPulled)
		res = append(res, line)
	}
	msg = fmt.Sprint(columnize.SimpleFormat(res))

	return noPushed, noPulled, msg, err
}
