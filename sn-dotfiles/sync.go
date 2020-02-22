package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"os"
	"strings"

	"github.com/ryanuber/columnize"
)

// Sync compares local and remote items and then:
// - pulls remotes if locals are older or missing
// - pushes locals if remotes are newer
func Sync(si SyncInput) (so SyncOutput, err error) {
	if err = checkPathsExist(si.Exclude); err != nil {
		return
	}

	var remote tagsWithNotes

	remote, err = get(si.Session, si.PageSize, si.Debug)
	if err != nil {
		return
	}

	err = checkNoteTagConflicts(remote)
	if err != nil {
		return
	}

	var sOut syncOutput
	sOut, err = sync(syncInput{session: si.Session, twn: remote, home: si.Home, paths: si.Paths,
		exclude: si.Exclude, debug: si.Debug})

	if err != nil {
		return
	}

	return SyncOutput{
		NoPushed: sOut.noPushed,
		NoPulled: sOut.noPulled,
		Msg:      sOut.msg,
	}, err
}

type SyncInput struct {
	Session        gosn.Session
	Home           string
	Paths, Exclude []string
	PageSize       int
	Debug          bool
}
type SyncOutput struct {
	NoPushed, NoPulled int
	Msg                string
}

func sync(si syncInput) (so syncOutput, err error) {
	var itemDiffs []ItemDiff

	itemDiffs, err = compare(si.twn, si.home, si.paths, si.exclude, si.debug)
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
		if matchesPathsToExclude(si.home, itemDiff.homeRelPath, si.exclude) {
			debugPrint(si.debug, fmt.Sprintf("sync | excluding: %s", itemDiff.homeRelPath))
			continue
		}

		switch itemDiff.diff {
		case localNewer:
			//push
			debugPrint(si.debug, fmt.Sprintf("sync | local %s is newer", itemDiff.homeRelPath))
			itemDiff.remote.Content.SetText(itemDiff.local)
			itemsToPush = append(itemsToPush, itemDiff)
			itemsToSync = true
		case localMissing:
			// pull
			debugPrint(si.debug, fmt.Sprintf("sync | %s is missing", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		case remoteNewer:
			// pull
			debugPrint(si.debug, fmt.Sprintf("sync | remote %s is newer", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		}
	}

	// check items to sync
	if !itemsToSync {
		so.msg = fmt.Sprint(bold("nothing to do"))
		return
	}

	// push
	if len(itemsToPush) > 0 {
		_, err = push(si.session, itemsToPush, si.debug)
		so.noPushed = len(itemsToPush)

		if err != nil {
			return
		}
	}

	res := make([]string, len(itemsToPush))
	strPushed := green("pushed")
	strPulled := green("pulled")

	for i, pushItem := range itemsToPush {
		line := fmt.Sprintf("%s | %s", bold(addDot(pushItem.homeRelPath)), strPushed)
		res[i] = line
	}

	// pull
	if err = pull(itemsToPull); err != nil {
		return
	}

	so.noPulled = len(itemsToPull)

	for _, pullItem := range itemsToPull {
		line := fmt.Sprintf("%s | %s\n", bold(addDot(pullItem.homeRelPath)), strPulled)
		res = append(res, line)
	}

	so.msg = fmt.Sprint(columnize.SimpleFormat(res))

	return so, err
}

type syncInput struct {
	session        gosn.Session
	twn            tagsWithNotes
	home           string
	paths, exclude []string
	debug          bool
}

type syncOutput struct {
	noPushed, noPulled int
	msg                string
}

func ensureTrailingPathSep(in string) string {
	if strings.HasSuffix(in, string(os.PathSeparator)) {
		return in
	}

	return in + string(os.PathSeparator)
}

func matchesPathsToExclude(home, path string, pathsToExclude []string) bool {
	for _, pte := range pathsToExclude {
		homeStrippedPath := stripHome(pte, home)
		// return match if Paths match exactly
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
