package sndotfiles

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/items"
	snsession "github.com/jonhadfield/gosn-v2/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllTagsWithoutNotes(t *testing.T) {
	fiestaNote := createNote("fiesta", "")
	focusNote := createNote("focus", "")
	carsFordTag := createTag("cars.ford")

	twn := tagsWithNotes{
		tagWithNotes{tag: carsFordTag, notes: items.Notes{fiestaNote, focusNote}},
	}
	tagsWithoutNotes := getAllTagsWithoutNotes(twn, items.Notes{focusNote}, true)
	// should be zero as cars.ford tag still has fiesta note remaining
	assert.Len(t, tagsWithoutNotes, 0)
	tagsWithoutNotes = getAllTagsWithoutNotes(twn, items.Notes{focusNote, fiestaNote}, true)
	// should be one as cars.ford tag no longer has notes (function doesn't check if cars tag is empty)
	assert.Len(t, tagsWithoutNotes, 1)
}

func TestTagTitleToFSDIR(t *testing.T) {
	home := getTemporaryHome()
	// missing Home should return err
	p, err := tagTitleToFSDir(fmt.Sprintf("%s.fruit.lemon", DotFilesTag), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "home directory required")
	assert.Empty(t, p)

	// check result for supplied title and Home
	p, err = tagTitleToFSDir(DotFilesTag, home)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/", home), p)

	// missing title should generate error
	p, err = tagTitleToFSDir("", home)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag title required")
	assert.Equal(t, "", p)
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
	// matchCase must be honoured
	assert.False(t, StringInSlice("jane", []string{"Rod", "JAne", "Freddy"}, true))
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
	iDiff, err := compareNoteWithFile("apple", applePath, home, appleNote, true)
	assert.NoError(t, err)
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
	lemonNote.UpdatedAt = time.Now().Add(1 * time.Hour).Format(common.TimeLayout)
	lemonPath := fmt.Sprintf("%s/lemon", home)
	assert.NoError(t, err)
	var f *os.File
	f, err = os.Create(lemonPath)
	assert.NoError(t, err)
	_, err = f.WriteString("lemon content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote differ and remote newer produces correct ItemDiff
	iDiff, err := compareNoteWithFile("lemon", lemonPath, home, lemonNote, true)
	assert.NoError(t, err)
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
	lemonNote.UpdatedAt = time.Now().Add(-1 * time.Hour).Format(common.TimeLayout)
	lemonPath := fmt.Sprintf("%s/lemon", home)
	assert.NoError(t, err)
	var f *os.File
	f, err = os.Create(lemonPath)
	assert.NoError(t, err)
	_, err = f.WriteString("lemon content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote differ and local newer produces correct ItemDiff
	iDiff, err := compareNoteWithFile("lemon", lemonPath, home, lemonNote, true)
	assert.NoError(t, err)
	assert.Equal(t, localNewer, iDiff.diff)
	assert.Equal(t, "lemon", iDiff.tagTitle)
	assert.Equal(t, "lemon", iDiff.noteTitle)
	assert.Equal(t, lemonPath, iDiff.path)
	assert.Equal(t, lemonNote, iDiff.remote)
}

func TestStripDot(t *testing.T) {
	assert.Equal(t, "test", stripDot(".test"))
	assert.Equal(t, "test", stripDot("test"))
}

func TestIsUnEncryptedSession(t *testing.T) {
	assert.False(t, isUnencryptedSession("invalid"))
	assert.True(t, isUnencryptedSession("someone@example.com;https://sync.standardnotes.org;eyJhbGciOiJKUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c;8f0f5166841ca4dee2975c74cc7e0a4345ce24b54d7b215677a3d540303aa203;6d5ffc6f8e337e6e3ae6d0c3201d9e2d00ffee64672bc4fe1886ad31770c19f1"))
}

func TestParsesessionString(t *testing.T) {
	// ensure an invalid session returns an error, no email address, and an empty session
	email, sess, err := ParseSessionString("invalid session string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session invalid")
	assert.Empty(t, email)
	assert.NotNil(t, sess)
	assert.Equal(t, snsession.Session{}, sess)

	// ensure an invalid session returns an error, no email address, and an empty session
	email, sess, err = ParseSessionString("someone@example.com;https://sync.standardnotes.org;eyJhbGciOiJKUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c;8f0f5166841ca4dee2975c74cc7e0a4345ce24b54d7b215677a3d540303aa203;6d5ffc6f8e337e6e3ae6d0c3201d9e2d00ffee64672bc4fe1886ad31770c19f1")
	assert.NoError(t, err)
	assert.Equal(t, "someone@example.com", email)
	assert.NotNil(t, sess)
	assert.Equal(t, snsession.Session{Server: "https://sync.standardnotes.org",
		Token: "eyJhbGciOiJKUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
		sess)
}

func TestNoteWithTagExists(t *testing.T) {
	note := createNote("apple", "")
	tag := createTag("fruit")
	twn := tagsWithNotes{
		tagWithNotes{tag: tag, notes: items.Notes{note}},
	}
	assert.Equal(t, 1, noteWithTagExists("fruit", "apple", twn))
}

func TestPushNoItems(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	// get populated db
	si := cache.SyncInput{
		Session: testCacheSession,
		Close:   false,
	}
	cso, err := cache.Sync(si)
	require.NoError(t, err)

	err = addToDB(cso.DB, testCacheSession, []ItemDiff{}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no items")
}
