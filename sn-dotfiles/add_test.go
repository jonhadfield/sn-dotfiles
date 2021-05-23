package sndotfiles

import (
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/lithammer/shortuuid"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func removeDB(dbPath string) {
	if err := os.Remove(dbPath); err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			panic(err)
		}
	}
}

func CleanUp(session cache.Session) error {
	removeDB(session.CacheDBPath)
	err := gosn.DeleteContent(&gosn.Session{
		Token:             testCacheSession.Token,
		MasterKey:         testCacheSession.MasterKey,
		Server:            testCacheSession.Server,
		AccessToken:       testCacheSession.AccessToken,
		AccessExpiration:  testCacheSession.AccessExpiration,
		RefreshExpiration: testCacheSession.RefreshExpiration,
		RefreshToken:      testCacheSession.RefreshToken,
	})
	return err
}

func getTemporaryHome() string {
	home := fmt.Sprintf("%s/%s", os.TempDir(), shortuuid.New())
	return strings.ReplaceAll(home, "//", "/")
}

func TestAddNoPaths(t *testing.T) {
	ai := AddInput{
		Session: testCacheSession,
		Home:    getTemporaryHome(),
		Paths:   nil,
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
	ai := AddInput{Session: &cache.Session{
		Session:     nil,
		CacheDB:     nil,
		CacheDBPath: "",
	}, Home: home, Paths: []string{gitConfigPath}}
	_, err := Add(ai)
	assert.Error(t, err)
}

func TestAddInvalidPath(t *testing.T) {
	var err error
	defer func() {
		if err = CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"
	duffPath := fmt.Sprintf("%s/.invalid/dodgy", home)

	assert.NoError(t, createTemporaryFiles(fwc))

	ai := AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath, duffPath}}
	var ao AddOutput
	ao, err = Add(ai)

	assert.Error(t, err)
	assert.Equal(t, 0, len(ao.PathsAdded))
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddOne(t *testing.T) {
	var err error
	defer func() {
		if err = CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := getTemporaryHome()

	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add item
	ai := AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath}}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ao.PathsAdded))
	assert.Equal(t, applePath, ao.PathsAdded[0])
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddTwoSameTag(t *testing.T) {
	var err error
	defer func() {
		if err = CleanUp(*testCacheSession); err != nil {
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
	ai := AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath, vwPath, bananaPath}}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ao.PathsAdded))
	assert.Contains(t, ao.PathsAdded, applePath)
	assert.Contains(t, ao.PathsAdded, bananaPath)
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddRecursive(t *testing.T) {
	var err error
	defer func() {
		if err = CleanUp(*testCacheSession); err != nil {
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
	ai := AddInput{Session: testCacheSession, Home: home, Paths: []string{fruitPath, carsPath}}
	var ao AddOutput
	ao, err = Add(ai)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(ao.PathsAdded))
	assert.Contains(t, ao.PathsAdded, applePath)
	assert.Equal(t, 0, len(ao.PathsExisting))
	assert.Equal(t, 0, len(ao.PathsInvalid))
}

func TestAddAll(t *testing.T) {
	var err error
	defer func() {
		if err = CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
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
	ai := AddInput{Session: testCacheSession, Home: home, All: true}
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
