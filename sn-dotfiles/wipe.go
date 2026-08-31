package sndotfiles

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/briandowns/spinner"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/items"
)

func WipeDotfileTagsAndNotes(sess *cache.Session, pageSize int, useStdErr bool) (int, error) {
	// validate session
	if !sess.Valid() {
		return 0, errors.New("invalid session")
	}

	if !sess.Debug {
		prefix := HiWhite("syncing ")
		if _, err := os.Stat(sess.CacheDBPath); os.IsNotExist(err) {
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
		Session: sess,
		Close:   false,
	}

	cso, err := cache.Sync(si)
	if err != nil {
		return 0, err
	}

	itemsToRemove, err := wipeCacheDB(cso.DB, sess)

	// The db holds an exclusive lock on the cache file, so it has to be closed
	// before syncing changes back to SN. wipeCacheDB saves with close set, so
	// it is usually already closed by now; closing it twice is a no-op.
	if cErr := cso.DB.Close(); cErr != nil {
		debugPrint(sess.Debug, fmt.Sprintf("WipeDotfileTagsAndNotes | closing db: %s", cErr))
	}

	sess.CacheDB = nil

	if err != nil {
		return 0, err
	}

	if itemsToRemove == 0 {
		return 0, nil
	}

	// persist the deletions back to SN
	si.Close = true

	if _, err = cache.Sync(si); err != nil {
		return 0, err
	}

	return itemsToRemove, nil
}

// wipeCacheDB marks every dotfiles tag and note in the cache db as deleted and
// returns the number of items marked.
func wipeCacheDB(db *storm.DB, sess *cache.Session) (int, error) {
	remote, err := getTagsWithNotes(db, sess)
	if err != nil {
		return 0, err
	}

	var itemsToRemove items.Items

	for _, twn := range remote {
		tag := twn.tag
		tag.Deleted = true
		itemsToRemove = append(itemsToRemove, &tag)

		for n := range twn.notes {
			twn.notes[n].Deleted = true
			itemsToRemove = append(itemsToRemove, &twn.notes[n])
		}
	}

	debugPrint(sess.Debug, fmt.Sprintf("WipeDotfileTagsAndNotes | removing %d items", len(itemsToRemove)))

	if len(itemsToRemove) == 0 {
		return 0, nil
	}

	sess.CacheDB = db

	if err = cache.SaveItems(sess, db, itemsToRemove, true); err != nil {
		return 0, err
	}

	return len(itemsToRemove), nil
}
