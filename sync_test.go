package sndotfiles

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jonhadfield/gosn"
	"github.com/lithammer/shortuuid"
	"github.com/stretchr/testify/assert"
)

func TestSyncNoItems(t *testing.T) {
	session, err := getSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	home := fmt.Sprintf("%s/%s", os.TempDir(), shortuuid.New())
	// add item
	var noPushed, noPulled int
	noPushed, noPulled, err = Sync(session, home, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no remote dotfiles found")
	assert.Equal(t, 0, noPushed)
	assert.Equal(t, 0, noPulled)
}

func TestSync(t *testing.T) {
	session, err := getSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := fmt.Sprintf("%s/%s", os.TempDir(), shortuuid.New())

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"
	//golfPath := fmt.Sprintf("%s/.cars/vw/golf.txt", home)

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{applePath, yellowPath, premiumPath}, true)
	assert.NoError(t, err)
	assert.Len(t, added, 3)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
	//assert.Contains(t, missing, golfPath)

	// Sync with no changes
	var noPushed, noPulled int
	var twn tagsWithNotes
	twn, err = get(session)
	noPushed, noPulled, err = sync(session, twn, home, true)
	assert.NoError(t, err)
	assert.Equal(t, 0, noPushed)
	assert.Equal(t, 0, noPulled)

	// Sync with changes to push
	// update local apple file
	fwc[applePath] = "new apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	noPushed, noPulled, err = sync(session, twn, home, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, noPushed)
	assert.Equal(t, 0, noPulled)

	// Sync with changes to pull
	// update local apple file
	// update apple note
	updateTime := time.Now().Add(time.Minute * 10)
	var uTwn tagsWithNotes
	for _, x := range twn {
		if x.tag.Content.GetTitle() == "dotfiles.fruit" {
			var nnotes gosn.Items
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
	noPushed, noPulled, err = sync(session, uTwn, home, true)
	assert.NoError(t, err)
	assert.Equal(t, 0, noPushed)
	assert.Equal(t, 1, noPulled)

}
