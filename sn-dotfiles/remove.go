package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/jonhadfield/gosn-v2"

	"github.com/ryanuber/columnize"
)

type RemoveInput struct {
	Session  gosn.Session
	Home     string
	Paths    []string
	PageSize int
	Debug    bool
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

	if StringInSlice(ri.Home, []string{"/", "/home"}, true) {
		err = fmt.Errorf("not a good idea to use '%s' as home dir", ri.Home)
		return
	}

	// check paths defined
	if len(ri.Paths) == 0 {
		return ro, errors.New("paths not defined")
	}

	// remove any duplicate paths
	ri.Paths = dedupe(ri.Paths)
	debugPrint(ri.Debug, fmt.Sprintf("Remove | paths after dedupe: %d", len(ri.Paths)))

	// check paths are valid
	if err = checkFSPaths(ri.Paths); err != nil {
		return
	}

	var tagsWithNotes tagsWithNotes
	tagsWithNotes, err = get(ri.Session, ri.PageSize, ri.Debug)

	if err != nil {
		return
	}

	err = checkNoteTagConflicts(tagsWithNotes)
	if err != nil {
		return
	}

	var results []string

	var notesToRemove gosn.Notes

	for _, path := range ri.Paths {
		homeRelPath, pathsToRemove, matchingItems := getNotesToRemove(path, ri.Home, tagsWithNotes, ri.Debug)

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
	for _, n := range notesToRemove {
		debugPrint(ri.Debug, fmt.Sprintf("Remove | remove note: %s", n.Content.Title))
	}

	// find any empty tags to delete
	emptyTags := findEmptyTags(tagsWithNotes, notesToRemove, ri.Debug)

	// dedupe any tags to remove
	if emptyTags != nil {
		emptyTags.DeDupe()
	}

	for x, et := range emptyTags {
		debugPrint(ri.Debug, fmt.Sprintf("Remove | tags to remove: [%d] %s", x, et.Content.GetTitle()))
	}

	for x, n := range notesToRemove {
		debugPrint(ri.Debug, fmt.Sprintf("Remove | notes to remove: [%d] %s", x, n.Content.GetTitle()))
	}

	var a gosn.Items

	for i := range notesToRemove {
		a = append(a, &notesToRemove[i])
	}


	for i := range emptyTags {
		a = append(a, &emptyTags[i])
	}

	x := removeInput{items: a, session: ri.Session, debug: ri.Debug}
	if err = remove(x); err != nil {
		return
	}

	ro.Msg = fmt.Sprint(columnize.SimpleFormat(results))
	ro.NotesRemoved = len(notesToRemove)
	ro.TagsRemoved = len(emptyTags)

	return ro, err
}

type removeInput struct {
	session gosn.Session
	items   gosn.Items
	debug   bool
}

func remove(input removeInput) error {
	var err error

	var items gosn.Items

	for _, i := range input.items {
		debugPrint(true, fmt.Sprintf("Setting %s %v to be deleted", i.GetContentType(), i.GetContent()))
		i.SetDeleted(true)
		items = append(items, i)
	}

	if items == nil {
		return fmt.Errorf("no items to remove")
	}

	var pio gosn.SyncOutput

	pio, err = putItems(putItemsInput{
		session: input.session,
		items:   items,
		debug:   input.debug,
	})
	if err != nil {
		return err
	}

	debugPrint(input.debug, fmt.Sprintf("remove | items put: %d", len(pio.SavedItems)))

	return err
}

type putItemsInput struct {
	items   gosn.Items
	session gosn.Session
	debug   bool
}
