package sndotfiles

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTagTitleToFSDIR(t *testing.T) {
	home := getTemporaryHome()
	// missing Home should return err
	p, isHome, err := tagTitleToFSDIR(fmt.Sprintf("%s.fruit.lemon", DotFilesTag), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Home directory required")
	assert.False(t, isHome)
	assert.Empty(t, p)

	// check result for supplied title and Home
	p1, isHome1, err := tagTitleToFSDIR(DotFilesTag, home)
	assert.NoError(t, err)
	assert.True(t, isHome1)
	assert.Equal(t, fmt.Sprintf("%s/", home), p1)

	// missing title should generate error
	p2, isHome2, err := tagTitleToFSDIR("", home)
	assert.Error(t, err)
	assert.False(t, isHome2)
	assert.Contains(t, err.Error(), "tag title required")
	assert.Equal(t, "", p2)
}

func TestDeDupe(t *testing.T) {
	noDupes := dedupe([]string{"lemon", "apple", "grapefruit"})
	assert.Len(t, noDupes, 3)
	assert.Contains(t, noDupes, "lemon")
	assert.Contains(t, noDupes, "apple")
	assert.Contains(t, noDupes, "grapefruit")

	deDuped := dedupe([]string{"lemon", "apple", "grapefruit", "apple", "lemon", "pineapple"})
	assert.Len(t, deDuped, 4)
	assert.Contains(t, deDuped, "lemon")
	assert.Contains(t, deDuped, "apple")
	assert.Contains(t, deDuped, "grapefruit")
	assert.Contains(t, deDuped, "pineapple")

	emptyList := dedupe([]string{})
	assert.Len(t, emptyList, 0)
}

func TestCreateTag(t *testing.T) {
	newTag := createTag("my.test.tag")
	assert.Equal(t, "my.test.tag", newTag.Content.GetTitle())
	assert.Equal(t, "Tag", newTag.ContentType)
	assert.NotEmpty(t, newTag.UUID)
}

func TestStripHome(t *testing.T) {
	home := getTemporaryHome()
	h1 := stripHome(fmt.Sprintf("%s/my/path", home), home)
	assert.Equal(t, "my/path", h1)
	h2 := stripHome("/my/path", home)
	assert.Equal(t, "/my/path", h2)
	h3 := stripHome("", "")
	assert.Equal(t, "", h3)
}

func TestStringInSlice(t *testing.T) {
	assert.True(t, StringInSlice("JAne", []string{"Rod", "JAne", "Freddy"}, true))
	assert.True(t, StringInSlice("FrEddy", []string{"Rod", "Jane", "Freddy"}, false))
	assert.False(t, StringInSlice("Rod", []string{}, false))
	assert.True(t, StringInSlice("", []string{"hello", "", "world"}, true))
}

func TestCompareIdentical(t *testing.T) {
	home := getTemporaryHome()
	err := os.MkdirAll(home, os.ModePerm)
	// setup
	appleNote := createNote("apple", "apple content")
	applePath := fmt.Sprintf("%s/apple", home)
	assert.NoError(t, err)
	var f *os.File
	f, err = os.Create(applePath)
	assert.NoError(t, err)
	_, err = f.WriteString("apple content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote identical produces correct ItemDiff
	iDiff := compare("apple", applePath, home, appleNote, true)
	assert.Equal(t, identical, iDiff.diff)
	assert.Equal(t, "apple", iDiff.tagTitle)
	assert.Equal(t, "apple", iDiff.noteTitle)
	assert.Equal(t, applePath, iDiff.path)
	assert.Equal(t, appleNote, iDiff.remote)
}

func TestCompareRemoteNewer(t *testing.T) {
	home := getTemporaryHome()
	err := os.MkdirAll(home, os.ModePerm)
	// setup
	lemonNote := createNote("lemon", "lemon content 2")
	lemonNote.UpdatedAt = time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04:05.000Z")
	lemonPath := fmt.Sprintf("%s/lemon", home)
	assert.NoError(t, err)
	var f *os.File
	f, err = os.Create(lemonPath)
	assert.NoError(t, err)
	_, err = f.WriteString("lemon content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote differ and remote newer produces correct ItemDiff
	iDiff := compare("lemon", lemonPath, home, lemonNote, true)
	assert.Equal(t, remoteNewer, iDiff.diff)
	assert.Equal(t, "lemon", iDiff.tagTitle)
	assert.Equal(t, "lemon", iDiff.noteTitle)
	assert.Equal(t, lemonPath, iDiff.path)
	assert.Equal(t, lemonNote, iDiff.remote)
}
func TestCompareLocalNewer(t *testing.T) {
	home := getTemporaryHome()
	err := os.MkdirAll(home, os.ModePerm)
	// setup
	lemonNote := createNote("lemon", "lemon content 2")
	lemonNote.UpdatedAt = time.Now().Add(-1 * time.Hour).Format("2006-01-02T15:04:05.000Z")
	lemonPath := fmt.Sprintf("%s/lemon", home)
	assert.NoError(t, err)
	var f *os.File
	f, err = os.Create(lemonPath)
	assert.NoError(t, err)
	_, err = f.WriteString("lemon content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote differ and local newer produces correct ItemDiff
	iDiff := compare("lemon", lemonPath, home, lemonNote, true)
	assert.Equal(t, localNewer, iDiff.diff)
	assert.Equal(t, "lemon", iDiff.tagTitle)
	assert.Equal(t, "lemon", iDiff.noteTitle)
	assert.Equal(t, lemonPath, iDiff.path)
	assert.Equal(t, lemonNote, iDiff.remote)
}
