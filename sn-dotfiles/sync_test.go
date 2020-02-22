package sndotfiles

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSyncInvalidSession(t *testing.T) {
	_, err := Sync(SyncInput{
		Session: gosn.Session{
			Token:  "invalid",
			Mk:     "invalid",
			Ak:     "invalid",
			Server: "invalid",
		},
		Home:    getTemporaryHome(),
		Paths:   []string{},
		Exclude: []string{},
		Debug:   true,
	})
	assert.Error(t, err)
}

func TestSyncNoItems(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	home := getTemporaryHome()
	// add item
	var so SyncOutput
	so, err = Sync(SyncInput{
		Session: session,
		Home:    home,
		Paths:   []string{},
		Exclude: []string{},
		Debug:   true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no remote dotfiles found")
	assert.Equal(t, 0, so.NoPushed)
	assert.Equal(t, 0, so.NoPulled)
}

func TestSync(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, DefaultPageSize, true); err != nil {
			fmt.Println("failed to WipeTheLot", err)
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

	// Sync with changes to pull based on missing local
	var noPushed, noPulled int
	debugPrint(true, "test | sync with changes to pull based on missing local")
	var so syncOutput
	so, err = sync(syncInput{
		session: session,
		twn:     twn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, so.noPushed)
	assert.Equal(t, 1, so.noPulled)

	// Sync with changes to push
	debugPrint(true, "test | sync with single local content update")
	// update local apple file
	// wait a second so file is noticeably newer
	time.Sleep(2 * time.Second)
	fwc[applePath] = "new apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	so, err = sync(syncInput{
		session: session,
		twn:     twn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, so.noPushed)
	assert.Equal(t, 0, so.noPulled)

	// Sync with changes to pull
	debugPrint(true, "test | sync with changes to pull based on time")
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
	so, err = sync(syncInput{
		session: session,
		twn:     uTwn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, so.noPushed)
	assert.Equal(t, 1, so.noPulled)

	// Sync with nothing to do
	debugPrint(true, "test | sync with nothing to do")
	so, err = sync(syncInput{
		session: session,
		twn:     uTwn,
		home:    home,
		paths:   []string{},
		exclude: []string{},
		debug:   true,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, noPushed)
	assert.Equal(t, 0, noPulled)

}

func TestSyncWithExcludeAbsolutePaths(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, DefaultPageSize, true); err != nil {
			fmt.Println("failed to WipeTheLot", err)
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

	debugPrint(true, "test | sync with three changes to pull based on exclusion of golf path")
	golfPath := fmt.Sprintf("%s/.cars/vw/golf.txt", home)
	var so syncOutput
	so, err = sync(syncInput{
		session: session,
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
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, DefaultPageSize, true); err != nil {
			fmt.Println("failed to WipeTheLot", err)
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

	debugPrint(true, "test | sync with two changes to pull based on exclusion of cars path")
	carsPath := fmt.Sprintf("%s/.cars", home)
	var so syncOutput
	so, err = sync(syncInput{
		session: session,
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
