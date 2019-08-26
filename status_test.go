package sndotfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/jonhadfield/gosn"

	"github.com/stretchr/testify/assert"
)

func TestStatusInvalidSession(t *testing.T) {
	_, _, err := Status(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, getTemporaryHome(), []string{}, true)
	assert.Error(t, err)
}

func TestStatus(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, applePath, yellowPath, premiumPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 4)
	assert.Len(t, ao.PathsExisting, 0)
	assert.Len(t, ao.PathsInvalid, 0)
	var diffs []ItemDiff

	diffs, _, err = Status(session, home, []string{gitConfigPath, applePath, yellowPath, premiumPath}, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 4)
	var pDiff int
	for _, d := range diffs {
		switch d.path {
		case gitConfigPath:
			assert.Equal(t, ".gitconfig", d.noteTitle)
			assert.Equal(t, gitConfigPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case applePath:
			assert.Equal(t, "apple", d.noteTitle)
			assert.Equal(t, applePath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case yellowPath:
			assert.Equal(t, "yellow", d.noteTitle)
			assert.Equal(t, yellowPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case premiumPath:
			assert.Equal(t, "premium", d.noteTitle)
			assert.Equal(t, premiumPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		}
	}
	assert.Equal(t, 4, pDiff)
}

// TODO: use 'small' status and pass pre-gen twn **********
func testStatusSetup(home string) (twn tagsWithNotes, fwc map[string]string) {
	fruitTag := createTag("dotfiles.sn-dotfiles-test-fruit")
	appleNote := createNote("apple", "apple content")
	lemonNote := createNote("lemon", "lemon content")
	grapeNote := createNote("grape", "grape content")
	fruitTagWithNotes := tagWithNotes{tag: fruitTag, notes: gosn.Items{appleNote, lemonNote, grapeNote}}
	twn = tagsWithNotes{fruitTagWithNotes}

	fwc = make(map[string]string)
	fwc[fmt.Sprintf("%s/.sn-dotfiles-test-fruit/apple", home)] = "apple content"
	fwc[fmt.Sprintf("%s/.sn-dotfiles-test-fruit/lemon", home)] = "lemon content"
	return
}

func TestStatus1(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	awsConfig := fmt.Sprintf("%s/.aws/config", home)
	fwc[awsConfig] = "aws config content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, awsConfig}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 2)
	assert.Len(t, ao.PathsExisting, 0)
	assert.Len(t, ao.PathsInvalid, 0)
	var diffs []ItemDiff

	diffs, _, err = Status(session, home, []string{gitConfigPath}, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 1)
	assert.Equal(t, ".gitconfig", diffs[0].noteTitle)
	assert.Equal(t, gitConfigPath, diffs[0].path)
	assert.Equal(t, identical, diffs[0].diff)
}

func TestStatus2(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, applePath, yellowPath, premiumPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 4)
	assert.Len(t, ao.PathsExisting, 0)
	assert.Len(t, ao.PathsInvalid, 0)
	var diffs []ItemDiff

	// delete apple so that a local item is missing
	err = os.Remove(applePath)
	assert.NoError(t, err)

	// update yellow content
	d1 := []byte("new yellow content")
	// pause so that local updated time newer
	time.Sleep(1 * time.Second)
	assert.NoError(t, ioutil.WriteFile(yellowPath, d1, 0644))

	// create untracked file
	d1 = []byte("green content")
	greenPath := fmt.Sprintf("%s/.fruit/banana/green", home)
	assert.NoError(t, ioutil.WriteFile(greenPath, d1, 0644))

	// update premium remote to trigger remote newer condition
	assert.NoError(t, updateRemoteNote(session, "premium", "new content"))

	diffs, _, err = Status(session, home, []string{fmt.Sprintf("%s/.fruit", home), fmt.Sprintf("%s/.cars", home)}, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 4)
	var pDiff int
	for _, d := range diffs {
		switch d.path {
		case gitConfigPath:
			assert.Equal(t, ".gitconfig", d.noteTitle)
			assert.Equal(t, gitConfigPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case applePath:
			assert.Equal(t, "apple", d.noteTitle)
			assert.Equal(t, applePath, d.path)
			assert.Equal(t, localMissing, d.diff)
			pDiff++
		case yellowPath:
			assert.Equal(t, "yellow", d.noteTitle)
			assert.Equal(t, yellowPath, d.path)
			assert.Equal(t, localNewer, d.diff)
			pDiff++
		case greenPath:
			assert.Equal(t, greenPath, d.path)
			assert.Equal(t, untracked, d.diff)
			pDiff++
		case premiumPath:
			assert.Equal(t, "premium", d.noteTitle)
			assert.Equal(t, premiumPath, d.path)
			assert.Equal(t, remoteNewer, d.diff)
			pDiff++
		}
	}
	assert.Equal(t, 4, pDiff)
}

func updateRemoteNote(session gosn.Session, noteTitle, newContent string) error {
	gii := gosn.GetItemsInput{Session: session}
	gio, err := gosn.GetItems(gii)
	if err != nil {
		return err
	}
	eItems := gio.Items
	eItems.DeDupe()
	var items gosn.Items
	items, err = eItems.DecryptAndParse(session.Mk, session.Ak)
	var uItem gosn.Item
	for i := range items {
		if items[i].ContentType == "Note" && items[i].Content.GetTitle() == noteTitle {
			items[i].Content.SetText(newContent)
			uItem = items[i]
			break
		}
	}
	nItems := gosn.Items{uItem}
	var eNItems gosn.EncryptedItems
	eNItems, err = nItems.Encrypt(session.Mk, session.Ak)
	pii := gosn.PutItemsInput{Session: session, Items: eNItems}
	_, err = gosn.PutItems(pii)
	return err
}
