package sndotfiles

import (
	"fmt"
	"os"
	"time"

	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/items"
)

func compare(remote tagsWithNotes, home string, paths, exclude []string, debug bool) (diffs []ItemDiff, err error) {
	debugPrint(debug, fmt.Sprintf("compare | Home: %s", home))
	debugPrint(debug, fmt.Sprintf("compare | %d Paths to include supplied", len(paths)))
	debugPrint(debug, fmt.Sprintf("compare | %d Paths to Exclude supplied", len(exclude)))
	// fail immediately if remote or Paths are empty
	if len(remote) == 0 {
		return nil, fmt.Errorf("tags with notes not supplied")
	}

	paths, err = preflight(home, paths)
	if err != nil {
		return
	}

	var itemDiffs []ItemDiff

	var remotePaths []string
	// check remotes against local filesystem
	itemDiffs, remotePaths, err = compareRemoteWithLocalFS(remote, paths, home, debug)
	if err != nil {
		return
	}

	// if Paths specified, then discover those that are untracked
	// by comparing with existing remote equivalent Paths
	if len(paths) > 0 {
		itemDiffs = append(itemDiffs, findUntracked(paths, remotePaths, home, debug)...)
	}

	return itemDiffs, err
}

func compareRemoteWithLocalFS(remote tagsWithNotes, paths []string, home string, debug bool) (itemDiffs []ItemDiff, remotePaths []string, err error) {
	// loop through remotes to generate a list of diffs for:
	// - existing local and remotes
	// - missing local files
	// also getTagsWithNotes a list of remotes that should have locals
	for _, twn := range remote {
		// only do a compare if path equals translated tag
		tagTitle := twn.tag.Content.GetTitle()

		var dir string

		dir, err = tagTitleToFSDir(twn.tag.Content.GetTitle(), home)
		if err != nil {
			return
		}

		debugPrint(debug, fmt.Sprintf("compare | tag title: %s is path: <home>/%s", tagTitle, stripHome(dir, home)))
		// if Paths were supplied, then check the determined dir is a prefix of one of those
		if len(paths) > 0 && !pathIsPrefixOfPaths(dir, paths) {
			continue
		}

		// loop through notes for the tag and compareNoteWithFile content of any with matching file
		// log each matching path so we can later walk them to discover untracked files
		for _, d := range twn.notes {
			fullPath := fmt.Sprintf("%s%s", dir, d.Content.GetTitle())
			// skip note if exact path is not specified and does not have prefix of total path
			if len(paths) > 0 && !noteInPaths(dir+d.Content.GetTitle(), paths) {
				continue
			}

			if !localExists(fullPath) {
				// local path matching tag+note doesn't exist so set as 'local missing'
				debugPrint(debug, fmt.Sprintf("compare | local not found: <home>/%s", stripHome(fullPath, home)))
				homeRelPath := stripHome(fullPath, home)

				itemDiffs = append(itemDiffs, ItemDiff{
					tagTitle:    tagTitle,
					homeRelPath: homeRelPath,
					path:        fullPath,
					diff:        localMissing,
					noteTitle:   d.Content.GetTitle(),
					remote:      d,
				})
			} else {
				// local does exist, so compareNoteWithFile and store generated compare
				debugPrint(debug, fmt.Sprintf("compare | local found: <home>/%s", stripHome(fullPath, home)))
				remotePaths = append(remotePaths, fullPath)

				var itemDiff ItemDiff

				itemDiff, err = compareNoteWithFile(tagTitle, fullPath, home, d, debug)
				if err != nil {
					return
				}

				itemDiffs = append(itemDiffs, itemDiff)
			}
		}
	}

	return itemDiffs, remotePaths, err
}

func compareNoteWithFile(tagTitle, path, home string, remote items.Note, debug bool) (ItemDiff, error) {
	debugPrint(debug, fmt.Sprintf("compareNoteWithFile | title: %s path: <home>/%s",
		tagTitle, stripHome(path, home)))

	localStat, err := os.Stat(path)
	if err != nil {
		return ItemDiff{}, err
	}

	localBytes, err := os.ReadFile(path)
	if err != nil {
		return ItemDiff{}, err
	}

	itemDiff := ItemDiff{
		tagTitle:    tagTitle,
		path:        path,
		homeRelPath: stripHome(path, home),
		noteTitle:   remote.Content.GetTitle(),
		diff:        identical,
		local:       string(localBytes),
		remote:      remote,
	}

	if itemDiff.local == remote.Content.GetText() {
		// local and remote identical
		return itemDiff, nil
	}

	remoteUpdated, err := time.Parse(common.TimeLayout, remote.UpdatedAt)
	if err != nil {
		return ItemDiff{}, fmt.Errorf("failed to parse updated time of note %q: %w", remote.Content.GetTitle(), err)
	}

	localUpdated := localStat.ModTime().UTC()

	debugPrint(debug, fmt.Sprintf("compareNoteWithFile | remote updated UTC): %v", remoteUpdated.UTC()))
	debugPrint(debug, fmt.Sprintf("compareNoteWithFile | local updated UTC): %v", localUpdated.Format(common.TimeLayout)))

	// if content different and local file was updated no earlier than the remote
	if !localUpdated.Before(remoteUpdated.UTC()) {
		itemDiff.diff = localNewer

		return itemDiff, nil
	}

	// content different and remote content was updated more recently
	itemDiff.diff = remoteNewer

	return itemDiff, nil
}
