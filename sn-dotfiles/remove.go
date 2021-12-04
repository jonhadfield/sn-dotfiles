package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/jonhadfield/gosn-v2"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/ryanuber/columnize"
	"os"
	"time"
)

type RemoveInput struct {
	Session  *cache.Session
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
func Remove(ri RemoveInput, useStdErr bool) (ro RemoveOutput, err error) {
	if StringInSlice(ri.Home, []string{"/", "/home"}, true) {
		err = fmt.Errorf("not a good idea to use '%s' as home dir", ri.Home)
		return
	}

	ri.Paths, err = preflight(ri.Home, ri.Paths)
	if err != nil {
		return
	}

	// check paths defined
	if len(ri.Paths) == 0 {
		return ro, errors.New("paths not defined")
	}

	// removeFromDB any duplicate paths
	ri.Paths = dedupe(ri.Paths)
	debugPrint(ri.Debug, fmt.Sprintf("Remove | paths after dedupe: %d", len(ri.Paths)))

	ri.Paths, err = preflight(ri.Home, ri.Paths)

	if !ri.Debug {
		prefix := HiWhite("syncing ")
		if _, err = os.Stat(ri.Session.CacheDBPath); os.IsNotExist(err) {
			prefix = HiWhite("initializing ")
		}

		s := spinner.New(spinner.CharSets[SpinnerCharSet], SpinnerDelay*time.Millisecond, spinner.WithWriter(os.Stdout))
		if useStdErr {
			s = spinner.New(spinner.CharSets[SpinnerCharSet], SpinnerDelay*time.Millisecond, spinner.WithWriter(os.Stderr))
		}

		s.Prefix = prefix
		s.Start()
		defer s.Stop()
	}

	// get populated db
	si := cache.SyncInput{
		Session: ri.Session,
		Close:   false,
	}

	var cso cache.SyncOutput

	cso, err = cache.Sync(si)
	if err != nil {
		return
	}

	var twn tagsWithNotes
	twn, err = getTagsWithNotes(cso.DB, ri.Session)
	if err != nil {
		return
	}

	err = checkNoteTagConflicts(twn)
	if err != nil {
		return
	}

	var results []string

	var notesToRemove gosn.Notes

	for _, path := range ri.Paths {
		homeRelPath, pathsToRemove, matchingItems := getNotesToRemove(path, ri.Home, twn, ri.Debug)

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

	// dedupe any notes to removeFromDB
	if notesToRemove != nil {
		notesToRemove.DeDupe()
	}
	for _, n := range notesToRemove {
		debugPrint(ri.Debug, fmt.Sprintf("Remove | removeFromDB note: %s", n.Content.Title))
	}

	// find any empty tags to delete
	emptyTags := findEmptyTags(twn, notesToRemove, ri.Debug)

	// dedupe any tags to removeFromDB
	if emptyTags != nil {
		emptyTags.DeDupe()
	}

	for x, et := range emptyTags {
		debugPrint(ri.Debug, fmt.Sprintf("Remove | tags to removeFromDB: [%d] %s", x, et.Content.GetTitle()))
	}

	for x, n := range notesToRemove {
		debugPrint(ri.Debug, fmt.Sprintf("Remove | notes to removeFromDB: [%d] %s", x, n.Content.GetTitle()))
	}

	var a gosn.Items

	for i := range notesToRemove {
		a = append(a, &notesToRemove[i])
	}

	for i := range emptyTags {
		a = append(a, &emptyTags[i])
	}
	ri.Session.CacheDB = cso.DB
	x := removeInput{items: a, session: ri.Session}
	if err = removeFromDB(x); err != nil {
		return
	}

	// sync changes back to SN
	si.Close = true
	cso, err = cache.Sync(si)
	if err != nil {
		return
	}

	ro.Msg = fmt.Sprint(columnize.SimpleFormat(results))
	ro.NotesRemoved = len(notesToRemove)
	ro.TagsRemoved = len(emptyTags)

	return ro, err
}

type removeInput struct {
	session *cache.Session
	items   gosn.Items
}

func removeFromDB(input removeInput) error {
	if !input.session.Valid() {
		return errors.New("session is invalid")
	}
	var items gosn.Items

	for _, i := range input.items {
		i.SetDeleted(true)
		items = append(items, i)
	}

	if items == nil {
		return fmt.Errorf("no items to removeFromDB")
	}

	var err error
	if err = cache.SaveItems(input.session.CacheDB, input.session, items, true); err != nil {
		return err
	}

	return err
}
