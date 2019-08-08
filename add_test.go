package sndotfiles

import (
	"fmt"
	"os"
	"testing"

	"github.com/lithammer/shortuuid"
	"github.com/stretchr/testify/assert"
)

func TestAddInvalidPath(t *testing.T) {
	session, err := getSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"
	duffPath := fmt.Sprintf("%s/.invalid/dodgy", home)

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{applePath, duffPath}, true, true)
	assert.Error(t, err)
	assert.Equal(t, 0, len(added))
	assert.Equal(t, 0, len(existing))
	assert.Equal(t, 0, len(missing))
}

func TestAddOne(t *testing.T) {
	session, err := getSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := fmt.Sprintf("%s%s", os.TempDir(), shortuuid.New())

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{applePath}, true, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(added))
	assert.Equal(t, applePath, added[0])
	assert.Equal(t, 0, len(existing))
	assert.Equal(t, 0, len(missing))
}

func TestAddTwoSameTag(t *testing.T) {
	session, err := getSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := fmt.Sprintf("%s%s", os.TempDir(), shortuuid.New())

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	vwPath := fmt.Sprintf("%s/.cars/vw", home)
	fwc[vwPath] = "vw content"
	bananaPath := fmt.Sprintf("%s/.fruit/banana", home)
	fwc[bananaPath] = "banana content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{applePath, vwPath, bananaPath}, true, true)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(added))
	assert.Contains(t, added, applePath)
	assert.Contains(t, added, bananaPath)
	assert.Equal(t, 0, len(existing))
	assert.Equal(t, 0, len(missing))

	var twn tagsWithNotes
	twn, err = get(session)
	assert.NoError(t, err)
	tagCount := len(twn)
	assert.Equal(t, 3, tagCount)
}

func TestAddRecursive(t *testing.T) {
	session, err := getSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := wipe(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := fmt.Sprintf("%s%s", os.TempDir(), shortuuid.New())

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"
	//golfPath := fmt.Sprintf("%s/.cars/vw/golf.txt", home)

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{applePath, yellowPath, premiumPath}, true, true)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(added))
	assert.Contains(t, added, applePath)
	assert.Equal(t, 0, len(existing))
	assert.Equal(t, 0, len(missing))
}
