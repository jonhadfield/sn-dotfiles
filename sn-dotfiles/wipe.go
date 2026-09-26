package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/jonhadfield/gosn-v2/cache"
	gosn "github.com/jonhadfield/gosn-v2/items"
	"os"
	"time"
)

func WipeDotfileTagsAndNotes(session *cache.Session, rootTag string, pageSize int, useStdErr bool) (int, error) {
	var err error
	if rootTag, err = ResolveRootTag(rootTag); err != nil {
		return 0, err
	}

	if !session.Valid() {
		return 0, errors.New("invalid session")
	}

	if !session.Debug {
		prefix := HiWhite("syncing ")
		if _, err := os.Stat(session.CacheDBPath); os.IsNotExist(err) {
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
		Session: session,
		Close:   false,
		// always fetch, as dotfiles may have changed on another machine since the last sync
		AlwaysSync: true,
	}

	var cso cache.SyncOutput
	cso, err = cache.Sync(si)
	if err != nil {
		return 0, err
	}

	var remote tagsWithNotes

	remote, err = getTagsWithNotes(cso.DB, session, rootTag)
	if err != nil {
		_ = cso.DB.Close()

		return 0, err
	}

	var itemsToRemove gosn.Items

	for _, twn := range remote {
		twn.tag.Deleted = true
		t := twn.tag
		itemsToRemove = append(itemsToRemove, &t)

		for n := range twn.notes {
			twn.notes[n].Deleted = true
			itemsToRemove = append(itemsToRemove, &twn.notes[n])
		}
	}

	debugPrint(session.Debug, fmt.Sprintf("WipeDotfileTagsAndNotes | removing %d items", len(itemsToRemove)))

	if len(itemsToRemove) == 0 {
		return 0, cso.DB.Close()
	}

	if err = cache.SaveItems(session, cso.DB, itemsToRemove, true); err != nil {
		_ = cso.DB.Close()

		return 0, err
	}

	pii := cache.SyncInput{
		Session: session,
		Close:   true,
	}

	_, err = cache.Sync(pii)
	if err != nil {
		return 0, err
	}

	return len(itemsToRemove), err
}
