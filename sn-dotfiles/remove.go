package sndotfiles

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jonhadfield/gosn"
	"github.com/ryanuber/columnize"
)

type RemoveInput struct {
	Session gosn.Session
	Home    string
	Paths   []string
	Debug   bool
}

type RemoveOutput struct {
	NotesRemoved, TagsRemoved, NotTracked int
	Msg                                   string
}

// Remove stops tracking local Paths by removing the related notes from SN
func Remove(ri RemoveInput) (ro RemoveOutput, err error) {
	// ensure home is passed
	if len(ri.Home) == 0 {
		err = errors.New("home undefined")
		return
	}

	// ensure home is passed
	if len(ri.Paths) == 0 {
		err = errors.New("paths undefined")
		return
	}

	// remove any duplicate Paths
	ri.Paths = dedupe(ri.Paths)

	// verify Paths before delete
	if err = checkPathsExist(ri.Paths); err != nil {
		return
	}

	var tagsWithNotes tagsWithNotes
	tagsWithNotes, err = get(ri.Session)

	if err != nil {
		return
	}

	err = checkNoteTagConflicts(tagsWithNotes)
	if err != nil {
		return
	}

	var results []string

	var notesToRemove gosn.Items

	for _, path := range ri.Paths {
		homeRelPath, pathsToRemove, matchingItems := getNotesToRemove(path, ri.Home, tagsWithNotes)

		debugPrint(ri.Debug, fmt.Sprintf("Remove | items matching path '%s': %d", path, len(matchingItems)))

		if len(matchingItems) == 0 {
			boldHomeRelPath := bold(stripTrailingSlash(homeRelPath))
			results = append(results, fmt.Sprintf("%s | %s", boldHomeRelPath, yellow("not tracked")))
			ro.NotTracked++

			continue
		}

		for _, ptr := range pathsToRemove {
			results = append(results, fmt.Sprintf("%s | %s", bold(ptr), green("removed")))
		}

		notesToRemove = append(notesToRemove, matchingItems...)
	}

	// dedupe any notes to remove
	if notesToRemove != nil {
		notesToRemove.DeDupe()
	}

	// find any empty tags to delete
	emptyTags := findEmptyTags(tagsWithNotes, notesToRemove, ri.Debug)

	// dedupe any tags to remove
	if emptyTags != nil {
		emptyTags.DeDupe()
	}

	// add empty tags to list of items to remove
	itemsToRemove := append(notesToRemove, emptyTags...)

	debugPrint(ri.Debug, fmt.Sprintf("Remove | items to remove: %d", len(itemsToRemove)))

	if err = remove(ri.Session, itemsToRemove, ri.Debug); err != nil {
		return
	}

	ro.Msg = fmt.Sprint(columnize.SimpleFormat(results))
	ro.NotesRemoved = len(notesToRemove)
	ro.TagsRemoved = len(emptyTags)

	return ro, err
}

func remove(session gosn.Session, items gosn.Items, debug bool) error {
	var err error

	var itemsToRemove gosn.Items

	for _, item := range items {
		item.Deleted = true
		itemsToRemove = append(itemsToRemove, item)
	}

	if itemsToRemove == nil {
		return fmt.Errorf("no items to remove")
	}

	var pio gosn.PutItemsOutput

	pio, err = putItems(session, itemsToRemove)
	if err != nil {
		return err
	}

	debugPrint(debug, fmt.Sprintf("remove | items put: %d", len(pio.ResponseBody.SavedItems)))

	return err
}

func stripTrailingSlash(in string) string {
	if strings.HasSuffix(in, "/") {
		return in[:len(in)-1]
	}

	return in
}
