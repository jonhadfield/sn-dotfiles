package sndotfiles

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jonhadfield/gosn"
	"github.com/ryanuber/columnize"
)

// Sync compares local and remote items and then:
// - pulls remotes if locals are older or missing
// - pushes locals if remotes are newer
func Sync(in SyncInput) (out SyncOutput, err error) {
	if err = checkPathsExist(in.Exclude); err != nil {
		return
	}

	var remote tagsWithNotes

	remote, err = get(in.Session)
	if err != nil {
		return
	}

	err = checkNoteTagConflicts(remote)
	if err != nil {
		return
	}

	var sOut syncOutput
	sOut, err = sync(syncInput{session: in.Session, twn: remote, home: in.Home, paths: in.Paths,
		exclude: in.Exclude, debug: in.Debug})

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
	Debug          bool
}
type SyncOutput struct {
	NoPushed, NoPulled int
	Msg                string
}

func sync(in syncInput) (out syncOutput, err error) {
	var itemDiffs []ItemDiff

	itemDiffs, err = compare(in.twn, in.home, in.paths, in.exclude, in.debug)
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
		if matchesPathsToExclude(in.home, itemDiff.homeRelPath, in.exclude) {
			debugPrint(in.debug, fmt.Sprintf("sync | excluding: %s", itemDiff.homeRelPath))
			continue
		}

		switch itemDiff.diff {
		case localNewer:
			//push
			debugPrint(in.debug, fmt.Sprintf("sync | local %s is newer", itemDiff.homeRelPath))
			itemDiff.remote.Content.SetText(itemDiff.local)
			itemsToPush = append(itemsToPush, itemDiff)
			itemsToSync = true
		case localMissing:
			// pull
			debugPrint(in.debug, fmt.Sprintf("sync | %s is missing", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		case remoteNewer:
			// pull
			debugPrint(in.debug, fmt.Sprintf("sync | remote %s is newer", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		}
	}

	// check items to sync
	if !itemsToSync {
		out.msg = fmt.Sprint(bold("nothing to do"))
		return
	}

	// push
	if len(itemsToPush) > 0 {
		_, err = push(in.session, itemsToPush)
		out.noPushed = len(itemsToPush)

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

	out.noPulled = len(itemsToPull)

	for _, pullItem := range itemsToPull {
		line := fmt.Sprintf("%s | %s\n", bold(addDot(pullItem.homeRelPath)), strPulled)
		res = append(res, line)
	}

	out.msg = fmt.Sprint(columnize.SimpleFormat(res))

	return out, err
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
