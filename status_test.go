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

func TestStatus(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
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
	var added, existing, missing []string
	added, existing, missing, _, err = Add(session, home, []string{gitConfigPath, applePath, yellowPath, premiumPath}, true)
	assert.NoError(t, err)
	assert.Len(t, added, 4)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
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
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
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
	var added, existing, missing []string
	added, existing, missing, _, err = Add(session, home, []string{gitConfigPath, awsConfig}, true)
	assert.NoError(t, err)
	assert.Len(t, added, 2)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
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
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
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
	var added, existing, missing []string
	added, existing, missing, _, err = Add(session, home, []string{gitConfigPath, applePath, yellowPath, premiumPath}, true)
	assert.NoError(t, err)
	assert.Len(t, added, 4)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
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

	diffs, _, err = Status(session, home, []string{fmt.Sprintf("%s/.fruit", home)}, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 3)
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
			assert.Equal(t, identical, d.diff)
			pDiff++
		}
	}
	assert.Equal(t, 3, pDiff)
}
