package sndotfiles

import (
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"github.com/jonhadfield/gosn-v2/cache"
)

func WipeDotfileTagsAndNotes(session *cache.Session, pageSize int, debug bool) (int, error) {
	// get populated db
	si := cache.SyncInput{
		Session: session,
		Close:   false,
	}

	var err error
	var cso cache.SyncOutput
	cso, err = cache.Sync(si)
	if err != nil {
		return 0, err
	}

	var remote tagsWithNotes

	remote, err = getTagsWithNotes(cso.DB, session)
	if err != nil {
		return 0, err
	}
	if err = cso.DB.Close(); err != nil {
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

	debugPrint(debug, fmt.Sprintf("WipeDotfileTagsAndNotes | removing %d items", len(itemsToRemove)))

	if len(itemsToRemove) == 0 {
		return 0, nil
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
