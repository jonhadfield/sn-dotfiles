package sndotfiles

import (
	"fmt"
	"os"
	"testing"

	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedRemote pushes items straight to the account, bypassing sn-dotfiles, so
// that tests can set up remote state the CLI would not itself create.
func seedRemote(t *testing.T, its items.Items) {
	t.Helper()

	so, err := cache.Sync(cache.SyncInput{Session: testCacheSession, Close: false})
	require.NoError(t, err)

	// SaveItems closes the db, which has to happen before the sync below can
	// reopen it.
	require.NoError(t, cache.SaveItems(testCacheSession, so.DB, its, true))

	testCacheSession.CacheDB = nil

	_, err = cache.Sync(cache.SyncInput{Session: testCacheSession, Close: true})
	require.NoError(t, err)

	testCacheSession.CacheDB = nil
}

// addFile writes a dotfile under home and tracks it.
func addFile(t *testing.T, home, name, content string) string {
	t.Helper()

	path := fmt.Sprintf("%s/%s", home, name)
	require.NoError(t, createTemporaryFiles(map[string]string{path: content}))

	ao, err := Add(AddInput{Session: testCacheSession, Home: home, Paths: []string{path}}, true)
	require.NoError(t, err)
	require.Len(t, ao.PathsAdded, 1)

	return path
}

// TestWipeDeletesRemotely covers wipe having previously reported items as
// removed while never actually saving the deletions.
func TestWipeDeletesRemotely(t *testing.T) {
	srv := requireMockServer(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	require.NoError(t, CleanUp(*testCacheSession))

	home := getTemporaryHome()
	addFile(t, home, ".gitconfig", "git config content")

	require.Len(t, srv.LiveItemsOfType(common.SNItemTypeNote), 1)
	require.NotEmpty(t, srv.LiveItemsOfType(common.SNItemTypeTag))

	num, err := WipeDotfileTagsAndNotes(testCacheSession, "", DefaultPageSize, true)
	require.NoError(t, err)
	assert.Positive(t, num)

	// the count wipe reports has to match what actually got deleted
	assert.Empty(t, srv.LiveItemsOfType(common.SNItemTypeNote))
	assert.Empty(t, srv.LiveItemsOfType(common.SNItemTypeTag))
}

// TestAddAlreadyTrackedPathIsNotAnError covers add having failed outright when
// every path given to it was already tracked.
func TestAddAlreadyTrackedPathIsNotAnError(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()
	path := addFile(t, home, ".gitconfig", "git config content")

	ao, err := Add(AddInput{Session: testCacheSession, Home: home, Paths: []string{path}}, true)
	require.NoError(t, err)

	assert.Empty(t, ao.PathsAdded)
	assert.Equal(t, []string{path}, ao.PathsExisting)
	assert.Contains(t, ao.Msg, "already tracked")
}

// TestRemoveUntrackedPathReportsNotTracked covers remove having returned an
// error instead of reporting the path as untracked when nothing matched.
func TestRemoveUntrackedPathReportsNotTracked(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()
	// track one file so the account has a dotfiles tag, then remove another
	addFile(t, home, ".gitconfig", "git config content")

	untracked := fmt.Sprintf("%s/.zshrc", home)
	require.NoError(t, createTemporaryFiles(map[string]string{untracked: "zsh config"}))

	ro, err := Remove(RemoveInput{
		Session: testCacheSession,
		Home:    home,
		Paths:   []string{untracked},
	}, true)
	require.NoError(t, err)

	assert.Equal(t, 1, ro.NotTracked)
	assert.Zero(t, ro.NotesRemoved)
	assert.Zero(t, ro.TagsRemoved)
	assert.Contains(t, ro.Msg, "not tracked")
}

// TestTagsMerelyContainingDotfilesAreIgnored covers the dotfiles tag pattern
// having been unanchored, which made any tag with "dotfiles" anywhere in its
// title look like a dotfiles tag.
func TestTagsMerelyContainingDotfilesAreIgnored(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	require.NoError(t, CleanUp(*testCacheSession))

	note, err := items.NewNote("notes-about-my-dotfiles", "not a dotfile", nil)
	require.NoError(t, err)

	// a tag whose title contains, but does not start with, "dotfiles"
	tag, err := items.NewTag("mydotfilesbackup", items.ItemReferences{{
		UUID:        note.UUID,
		ContentType: common.SNItemTypeNote,
	}})
	require.NoError(t, err)

	seedRemote(t, items.Items{&note, &tag})

	home := getTemporaryHome()
	require.NoError(t, os.MkdirAll(home, os.ModePerm))

	_, msg, err := Status(testCacheSession, home, []string{}, nil, "", DefaultPageSize, true, true)
	require.NoError(t, err)

	assert.Equal(t, "no dotfiles being tracked", msg)
}

// TestDotfilesDescendantTagsAreTracked is the counterpart to the test above:
// the dotfiles tag and its descendants must still be picked up.
func TestDotfilesDescendantTagsAreTracked(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	require.NoError(t, CleanUp(*testCacheSession))

	home := getTemporaryHome()
	require.NoError(t, createTemporaryFiles(map[string]string{
		fmt.Sprintf("%s/.gitconfig", home):       "git config content",
		fmt.Sprintf("%s/.aws/config", home):      "aws config content",
		fmt.Sprintf("%s/.aws/credentials", home): "aws credentials content",
	}))

	ao, err := Add(AddInput{
		Session: testCacheSession,
		Home:    home,
		Paths:   []string{fmt.Sprintf("%s/.gitconfig", home), fmt.Sprintf("%s/.aws", home)},
	}, true)
	require.NoError(t, err)
	require.Len(t, ao.PathsAdded, 3)

	diffs, _, err := Status(testCacheSession, home, []string{}, nil, "", DefaultPageSize, true, true)
	require.NoError(t, err)

	// the note under the root dotfiles tag and both under dotfiles.aws
	require.Len(t, diffs, 3)

	for _, d := range diffs {
		assert.Equal(t, identical, d.diff, d.homeRelPath)
	}
}

// TestContentRoundTripsThroughTheServer checks a tracked file's content
// survives encryption, the sync round trip and decryption, by pulling it back
// down into an empty home from a cold cache.
func TestContentRoundTripsThroughTheServer(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	require.NoError(t, CleanUp(*testCacheSession))

	const content = "[user]\n\tname = A Tester\n"

	home := getTemporaryHome()
	addFile(t, home, ".gitconfig", content)

	// a different home with no local copy, and a cache built from scratch
	restoreHome := getTemporaryHome()
	require.NoError(t, os.MkdirAll(restoreHome, os.ModePerm))
	removeDB(testCacheSession.CacheDBPath)

	so, err := Sync(SNDotfilesSyncInput{
		Session: testCacheSession,
		Home:    restoreHome,
		Debug:   true,
	}, true)
	require.NoError(t, err)

	assert.Equal(t, 1, so.NoPulled)
	assert.Zero(t, so.NoPushed)

	restored, err := os.ReadFile(fmt.Sprintf("%s/.gitconfig", restoreHome))
	require.NoError(t, err)
	assert.Equal(t, content, string(restored))
}

// TestStatusWarnsAboutEditorAssociations covers the check that replaced the
// long-standing TODO in sync: an editor that stores anything but plain text
// rewrites a dotfile when it saves, so the association is worth reporting.
func TestStatusWarnsAboutEditorAssociations(t *testing.T) {
	requireLiveSession(t)

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	home := getTemporaryHome()
	addFile(t, home, ".gitconfig", "git config content")

	// find the note the add created, so the component can claim it
	so, err := cache.Sync(cache.SyncInput{Session: testCacheSession, Close: false})
	require.NoError(t, err)

	var cached cache.Items
	require.NoError(t, so.DB.All(&cached))
	require.NoError(t, so.DB.Close())

	testCacheSession.CacheDB = nil

	all, err := cached.ToItems(testCacheSession)
	require.NoError(t, err)

	var noteUUID string

	for _, item := range all {
		if item.GetContentType() == common.SNItemTypeNote && item.GetContent() != nil {
			noteUUID = item.GetUUID()
		}
	}

	require.NotEmpty(t, noteUUID, "expected the added note to be in the account")

	component := items.NewComponent()
	content := items.NewComponentContent()
	content.Name = "Super"
	content.Area = editorArea
	content.AssociateItems([]string{noteUUID})
	component.Content = *content

	seedRemote(t, items.Items{&component})

	_, msg, err := Status(testCacheSession, home, []string{}, nil, "", DefaultPageSize, false, true)
	require.NoError(t, err)

	assert.Contains(t, msg, "an editor is associated with tracked dotfiles")
	assert.Contains(t, msg, ".gitconfig")
	assert.Contains(t, msg, "Super")
}
