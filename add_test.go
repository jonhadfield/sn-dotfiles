package sndotfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
)

//func GetSessionFromUser(server string) (gosn.Session, string, error) {
//	var sess gosn.Session
//	var email string
//	var err error
//	var password, apiServer, errMsg string
//	email, password, apiServer, errMsg = GetCredentials(server)
//	if errMsg != "" {
//		fmt.Printf("\nerror: %s\n\n", errMsg)
//		return sess, email, err
//	}
//	sess, err = gosn.CliSignIn(email, password, apiServer)
//	if err != nil {
//		return sess, email, err
//
//	}
//	return sess, email, err
//}

//func parseSessionString(in string) (email string, session gosn.Session) {
//	parts := strings.Split(in, ";")
//	email = parts[0]
//	session = gosn.Session{
//		Token:  parts[2],
//		Mk:     parts[4],
//		Ak:     parts[3],
//		Server: parts[1],
//	}
//	return
//}

func TestAddInvalidPath(t *testing.T) {
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
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"
	fruitPath := fmt.Sprintf("%s/.fruit", home)
	carsPath := fmt.Sprintf("%s/.cars", home)
	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{fruitPath, carsPath}, true, true)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(added))
	assert.Contains(t, added, applePath)
	assert.Equal(t, 0, len(existing))
	assert.Equal(t, 0, len(missing))
}

func TestCheckPathValid(t *testing.T) {
	home := getTemporaryHome()
	d1 := []byte("test file")
	filePath := home + "test"
	symLinkPath := home + "test_sym"
	assert.NoError(t, ioutil.WriteFile(filePath, d1, 0644))
	fmt.Println(1)
	assert.True(t, checkPathValid(filePath))
	fmt.Println(2)
	assert.NoError(t, os.Symlink(filePath, symLinkPath))
	fmt.Println(3)
	assert.False(t, checkPathValid(symLinkPath))
	fmt.Println(4)
	assert.False(t, false)
	fmt.Println(5)
}

func GetTestSession() (gosn.Session, error) {
	email := os.Getenv("SN_EMAIL")
	password := os.Getenv("SN_PASSWORD")
	apiServer := os.Getenv("SN_SERVER")
	return gosn.CliSignIn(email, password, apiServer)
}

func wipe(session gosn.Session) (int, error) {
	getItemsInput := gosn.GetItemsInput{
		Session: session,
	}
	var err error
	// get all existing Tags and Notes and mark for deletion
	var output gosn.GetItemsOutput
	output, err = gosn.GetItems(getItemsInput)
	if err != nil {
		return 0, err
	}
	output.Items.DeDupe()
	var pi gosn.Items
	pi, err = output.Items.DecryptAndParse(session.Mk, session.Ak)
	if err != nil {
		return 0, err
	}
	var itemsToDel gosn.Items
	for _, item := range pi {
		if item.Deleted {
			continue
		}

		switch {
		case item.ContentType == "Tag":
			item.Deleted = true
			item.Content = gosn.NewTagContent()
			itemsToDel = append(itemsToDel, item)
		case item.ContentType == "Note":
			item.Deleted = true
			item.Content = gosn.NewNoteContent()
			itemsToDel = append(itemsToDel, item)
		}
	}

	_, err = putItems(session, itemsToDel)
	if err != nil {
		return 0, err
	}
	return len(itemsToDel), err
}
