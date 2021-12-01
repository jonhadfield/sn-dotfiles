package sndotfiles

import (
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSyncInvalidSession(t *testing.T) {
	_, err := Sync(SNDotfilesSyncInput{
		Session: &cache.Session{
			Session:     nil,
			CacheDB:     nil,
			CacheDBPath: "",
		},
		Home:    getTemporaryHome(),
		Paths:   []string{},
		Exclude: []string{},
		Debug:   true,
	}, true)
	assert.Error(t, err)
}

func TestSyncNoItems(t *testing.T) {
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := getTemporaryHome()
	var err error
	// add item
	var so SyncOutput
	so, err = Sync(SNDotfilesSyncInput{
		Session: testCacheSession,
		Home:    home,
		Paths:   []string{},
		Exclude: []string{},
		Debug:   true,
	}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no remote dotfiles found")
	assert.Equal(t, 0, so.NoPushed)
	assert.Equal(t, 0, so.NoPulled)
}

// TestBasicSync adds a file to the remote, deletes the local file and then
// performs a sync to check it was added back
func TestBasicSync(t *testing.T) {
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	assert.NotEmpty(t, testCacheSession.AccessToken)
	home := getTemporaryHome()

	// add item
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	ai := AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath}}
	ao, err := Add(ai, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ao.PathsAdded))
	assert.Equal(t, applePath, ao.PathsAdded[0])
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))

	// delete local file so we can sync it back
	require.NoError(t, os.Remove(applePath))

	so, err := Sync(SNDotfilesSyncInput{
		Session: testCacheSession,
		Home:    home,
		Paths:   []string{},
		Exclude: []string{},
		Debug:   true,
	}, true)

	require.NoError(t, err)
	assert.Equal(t, 0, so.NoPushed)
	assert.Equal(t, 1, so.NoPulled)

	content, err := ioutil.ReadFile(applePath)
	require.NoError(t, err)
	require.Equal(t, "apple content", string(content))
}

// TestSyncTwoUpdatesFiles adds two files, updates them locally and syncs them back
func TestSyncTwoFilesUpdatedFiles(t *testing.T) {
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	assert.NotEmpty(t, testCacheSession.AccessToken)
	home := getTemporaryHome()

	// add item
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"
	lemonPath := fmt.Sprintf("%s/.lemon", home)
	fwc[lemonPath] = "lemon content"

	assert.NoError(t, createTemporaryFiles(fwc))
	ai := AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath, lemonPath}}
	ao, err := Add(ai, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ao.PathsAdded))
	assert.Equal(t, applePath, ao.PathsAdded[0])
	assert.Equal(t, lemonPath, ao.PathsAdded[1])
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
	assert.NoError(t, createPathWithContent(applePath, "apple content updated"))
	assert.NoError(t, createPathWithContent(lemonPath, "lemon content updated"))

	var so SyncOutput
	so, err = Sync(SNDotfilesSyncInput{
		Session: testCacheSession,
		Home:    home,
		Paths:   []string{applePath, lemonPath},
		Exclude: []string{},
		Debug:   true,
	}, true)

	require.NoError(t, err)
	require.Equal(t, so.NoPushed, 2)
}

// TestSync creates local dotfiles
func TestSync(t *testing.T) {
	assert.NotEmpty(t, testCacheSession.AccessToken)
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := getTemporaryHome()

	fruitTag := createTag("dotfiles.fruit")
	bananaTag := createTag("dotfiles.fruit.banana")
	carsTag := createTag("dotfiles.cars")
	vwTag := createTag("dotfiles.cars.vw")
	mercedesTag := createTag("dotfiles.cars.mercedes")
	a250Tag := createTag("dotfiles.cars.mercedes.a250")
	appleNote := createNote("apple", "apple content")
	yellowNote := createNote("yellow", "yellow content")
	golfNote := createNote("golf.txt", "golf content")
	premiumNote := createNote("premium", "premium content")

	fruitTagWithNotes := tagWithNotes{tag: fruitTag, notes: gosn.Notes{appleNote}}
	carsTagWithNotes := tagWithNotes{tag: carsTag, notes: gosn.Notes{}}
	bananaTagWithNotes := tagWithNotes{tag: bananaTag, notes: gosn.Notes{yellowNote}}
	vwTagWithNotes := tagWithNotes{tag: vwTag, notes: gosn.Notes{golfNote}}
	mercedesTagWithNotes := tagWithNotes{tag: mercedesTag, notes: gosn.Notes{}}
	a250TagWithNotes := tagWithNotes{tag: a250Tag, notes: gosn.Notes{premiumNote}}
	twn := tagsWithNotes{fruitTagWithNotes, carsTagWithNotes, bananaTagWithNotes, vwTagWithNotes, mercedesTagWithNotes, a250TagWithNotes}

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"

	assert.NoError(t, createTemporaryFiles(fwc))

	// get populated db
	si := cache.SyncInput{
		Session: testCacheSession,
		Close:   false,
	}
	cso, err := cache.Sync(si)
	require.NoError(t, err)

	// Sync with changes to createLocal based on missing local
	var noPushed, noPulled int
	debugPrint(true, "test | syncDBwithFS with changes to createLocal based on missing local")
	var so syncOutput
	so, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: testCacheSession,
		twn:     twn,
		home:    home,
		debug:   true,
		close:   false,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, so.noPushed)
	// .cars/vw/golf.txt should be pulled
	assert.Equal(t, 1, so.noPulled, ".cars/vw/golf.txt should have been pulled")
	assert.NoError(t, err)

	// Sync with changes to addToDB
	// update local apple file
	// wait a second so file is noticeably newer
	time.Sleep(1 * time.Second)
	fwc[applePath] = "new apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	so, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: testCacheSession,
		twn:     twn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
		close:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, so.noPushed)
	assert.Equal(t, 0, so.noPulled)

	// Sync with changes to createLocal
	debugPrint(true, "test | syncDBwithFS with changes to createLocal based on time")
	// update apple note
	updateTime := time.Now().UTC().Add(time.Minute * 10)
	var uTwn tagsWithNotes
	for _, x := range twn {
		if x.tag.Content.GetTitle() == "dotfiles.fruit" {
			var nnotes gosn.Notes
			for _, note := range x.notes {
				if note.Content.GetTitle() == "apple" {
					note.Content.SetText("new note content")
					note.Content.SetUpdateTime(updateTime)
					note.UpdatedAt = updateTime.Format("2006-01-02T15:04:05.000Z")
				}
				nnotes = append(nnotes, note)
				x.notes = nnotes
			}
		}
		uTwn = append(uTwn, x)
	}
	assert.NoError(t, err)
	so, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: testCacheSession,
		twn:     uTwn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
		close:   false,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, so.noPushed)
	assert.Equal(t, 1, so.noPulled)

	// Sync with nothing to do
	debugPrint(true, "test | syncDBwithFS with nothing to do")
	so, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: testCacheSession,
		twn:     uTwn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
		close:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, noPushed)
	assert.Equal(t, 0, noPulled)
}

func TestSyncWithExcludeAbsolutePaths(t *testing.T) {
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := getTemporaryHome()

	fruitTag := createTag("dotfiles.fruit")
	bananaTag := createTag("dotfiles.fruit.banana")
	carsTag := createTag("dotfiles.cars")
	vwTag := createTag("dotfiles.cars.vw")
	mercedesTag := createTag("dotfiles.cars.mercedes")
	a250Tag := createTag("dotfiles.cars.mercedes.a250")
	appleNote := createNote("apple", "apple content")
	yellowNote := createNote("yellow", "yellow content")
	golfNote := createNote("golf.txt", "golf content")
	premiumNote := createNote("premium", "premium content")

	fruitTagWithNotes := tagWithNotes{tag: fruitTag, notes: gosn.Notes{appleNote}}
	carsTagWithNotes := tagWithNotes{tag: carsTag, notes: gosn.Notes{}}
	bananaTagWithNotes := tagWithNotes{tag: bananaTag, notes: gosn.Notes{yellowNote}}
	vwTagWithNotes := tagWithNotes{tag: vwTag, notes: gosn.Notes{golfNote}}
	mercedesTagWithNotes := tagWithNotes{tag: mercedesTag, notes: gosn.Notes{}}
	a250TagWithNotes := tagWithNotes{tag: a250Tag, notes: gosn.Notes{premiumNote}}
	twn := tagsWithNotes{fruitTagWithNotes, carsTagWithNotes, bananaTagWithNotes, vwTagWithNotes, mercedesTagWithNotes, a250TagWithNotes}

	// get populated db
	si := cache.SyncInput{
		Session: testCacheSession,
		Close:   false,
	}
	cso, err := cache.Sync(si)
	require.NoError(t, err)

	debugPrint(true, "test | syncDBwithFS with three changes to createLocal based on exclusion of golf path")
	golfPath := fmt.Sprintf("%s/.cars/vw/golf.txt", home)

	var so syncOutput
	so, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: testCacheSession,
		twn:     twn,
		home:    home,
		paths:   []string{},
		exclude: []string{golfPath},
		debug:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, so.noPushed)
	assert.Equal(t, 3, so.noPulled)
}

func TestSyncWithExcludeParentPaths(t *testing.T) {
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()

	fruitTag := createTag("dotfiles.fruit")
	bananaTag := createTag("dotfiles.fruit.banana")
	carsTag := createTag("dotfiles.cars")
	vwTag := createTag("dotfiles.cars.vw")
	mercedesTag := createTag("dotfiles.cars.mercedes")
	a250Tag := createTag("dotfiles.cars.mercedes.a250")
	appleNote := createNote("apple", "apple content")
	yellowNote := createNote("yellow", "yellow content")
	golfNote := createNote("golf.txt", "golf content")
	premiumNote := createNote("premium", "premium content")

	fruitTagWithNotes := tagWithNotes{tag: fruitTag, notes: gosn.Notes{appleNote}}
	carsTagWithNotes := tagWithNotes{tag: carsTag, notes: gosn.Notes{}}
	bananaTagWithNotes := tagWithNotes{tag: bananaTag, notes: gosn.Notes{yellowNote}}
	vwTagWithNotes := tagWithNotes{tag: vwTag, notes: gosn.Notes{golfNote}}
	mercedesTagWithNotes := tagWithNotes{tag: mercedesTag, notes: gosn.Notes{}}
	a250TagWithNotes := tagWithNotes{tag: a250Tag, notes: gosn.Notes{premiumNote}}
	twn := tagsWithNotes{fruitTagWithNotes, carsTagWithNotes, bananaTagWithNotes, vwTagWithNotes, mercedesTagWithNotes, a250TagWithNotes}

	// get populated db
	si := cache.SyncInput{
		Session: testCacheSession,
		Close:   false,
	}
	cso, err := cache.Sync(si)
	assert.NoError(t, err)

	debugPrint(true, "test | syncDBwithFS with two changes to createLocal based on exclusion of cars path")
	carsPath := fmt.Sprintf("%s/.cars", home)
	var so syncOutput
	so, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: testCacheSession,
		twn:     twn,
		home:    home,
		paths:   []string{},
		exclude: []string{carsPath},
		debug:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, so.noPushed)
	assert.Equal(t, 2, so.noPulled)
}
