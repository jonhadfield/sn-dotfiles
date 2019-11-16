package sndotfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/lithammer/shortuuid"

	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	session, err := GetTestSession()
	if err != nil {
		fmt.Println("failed to get session:", err)
		os.Exit(1)
	}
	if _, err := WipeTheLot(session); err != nil {
		fmt.Println("failed to wipe:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func getTemporaryHome() string {
	home := fmt.Sprintf("%s/%s", os.TempDir(), shortuuid.New())
	return strings.ReplaceAll(home, "//", "/")
}

func TestAddNoPaths(t *testing.T) {
	ai := AddInput{
		Session: gosn.Session{},
		Home:    getTemporaryHome(),
		Paths:   nil,
		Debug:   true,
	}
	_, err := Add(ai)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "paths")
}

func TestAddInvalidSession(t *testing.T) {
	home := getTemporaryHome()
	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"

	assert.NoError(t, createTemporaryFiles(fwc))
	ai := AddInput{Session: gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, Home: home, Paths: []string{gitConfigPath}, Debug: true}
	_, err := Add(ai)
	assert.Error(t, err)
}

func TestAddInvalidPath(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeTheLot(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"
	duffPath := fmt.Sprintf("%s/.invalid/dodgy", home)

	assert.NoError(t, createTemporaryFiles(fwc))

	ai := AddInput{Session: session, Home: home, Paths: []string{applePath, duffPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)

	assert.Error(t, err)
	assert.Equal(t, 0, len(ao.PathsAdded))
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddOne(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeTheLot(session); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := getTemporaryHome()

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	ai := AddInput{Session: session, Home: home, Paths: []string{applePath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ao.PathsAdded))
	assert.Equal(t, applePath, ao.PathsAdded[0])
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddTwoSameTag(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeTheLot(session); err != nil {
			fmt.Println("failed to WipeTheLot")
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
	ai := AddInput{Session: session, Home: home, Paths: []string{applePath, vwPath, bananaPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ao.PathsAdded))
	assert.Contains(t, ao.PathsAdded, applePath)
	assert.Contains(t, ao.PathsAdded, bananaPath)
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))

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
		if _, err := WipeTheLot(session); err != nil {
			fmt.Println("failed to WipeTheLot")
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
	ai := AddInput{Session: session, Home: home, Paths: []string{fruitPath, carsPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ao.PathsAdded))
	assert.Contains(t, ao.PathsAdded, applePath)
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddAll(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeTheLot(session); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()

	fwc := make(map[string]string)
	file1Path := fmt.Sprintf("%s/.file1", home)
	fwc[file1Path] = "file1 content"
	file2Path := fmt.Sprintf("%s/.file2", home)
	fwc[file2Path] = "yellow content"
	file3Path := fmt.Sprintf("%s/file3", home)
	fwc[file3Path] = "file3 content"
	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	ai := AddInput{Session: session, Home: home, All: true, Debug: true}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ao.PathsAdded))
	assert.Contains(t, ao.PathsAdded, file1Path)
	assert.Contains(t, ao.PathsAdded, file2Path)
	assert.NotContains(t, ao.PathsAdded, file3Path)
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestCheckPathValid(t *testing.T) {
	home := getTemporaryHome()
	d1 := []byte("test file")
	filePath := home + "test"
	symLinkPath := home + "test_sym"
	assert.NoError(t, ioutil.WriteFile(filePath, d1, 0644))
	v, err := pathValid(filePath)
	assert.True(t, v)
	assert.NoError(t, err)
	assert.NoError(t, os.Symlink(filePath, symLinkPath))
	v, err = pathValid(symLinkPath)
	assert.False(t, v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlink")
	assert.False(t, false)
}

func TestCreateItemInvalidPath(t *testing.T) {
	_, err := createItem("invalid", "title")
	assert.Error(t, err)
}

func GetTestSession() (gosn.Session, error) {
	email := os.Getenv("SN_EMAIL")
	password := os.Getenv("SN_PASSWORD")
	apiServer := os.Getenv("SN_SERVER")
	return gosn.CliSignIn(email, password, apiServer)
}

func WipeTheLot(session gosn.Session) (int, error) {
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
