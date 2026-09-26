package sndotfiles

import (
	"fmt"
	"github.com/jonhadfield/gosn-v2/cache"
	gosn "github.com/jonhadfield/gosn-v2/items"
	"github.com/jonhadfield/gosn-v2/session"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetAllTagsWithoutNotes(t *testing.T) {
	fiestaNote := newTestNote()
	fiestaNoteContent := gosn.NewNoteContent()
	fiestaNoteContent.SetTitle("fiesta")
	fiestaNote.Content = *fiestaNoteContent

	focusNote := newTestNote()
	focusNoteContent := gosn.NewNoteContent()
	focusNoteContent.SetTitle("focus")
	focusNote.Content = *focusNoteContent

	carsTagContent := gosn.NewTagContent()
	carsTag := newTestTag()
	carsTagContent.SetTitle("cars")
	carsTag.Content = *carsTagContent

	carsFordTagContent := gosn.NewTagContent()
	carsFordTag := newTestTag()
	carsFordTagContent.SetTitle("cars.ford")
	carsFordTag.Content = *carsTagContent

	twn := tagsWithNotes{
		tagWithNotes{tag: carsFordTag, notes: gosn.Notes{fiestaNote, focusNote}},
	}
	tagsWithoutNotes := getAllTagsWithoutNotes(twn, gosn.Notes{focusNote}, true)
	// should be zero as cars.ford tag still has fiesta note remaining
	assert.Len(t, tagsWithoutNotes, 0)
	tagsWithoutNotes = getAllTagsWithoutNotes(twn, gosn.Notes{focusNote, fiestaNote}, true)
	// should be one as cars.ford tag no longer has notes (function doesn't check if cars tag is empty)
	assert.Len(t, tagsWithoutNotes, 1)
}

func TestTagTitleToFSDIR(t *testing.T) {
	home := getTemporaryHome()
	// missing Home should return err
	p, err := tagTitleToFSDir(fmt.Sprintf("%s.fruit.lemon", DotFilesTag), "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "home directory required")
	assert.Empty(t, p)

	// check result for supplied title and Home
	p, err = tagTitleToFSDir(DotFilesTag, home, "")
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/", home), p)

	// custom root tag
	p, err = tagTitleToFSDir("PersonalDotfiles.config.fish", home, "PersonalDotfiles")
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/.config/fish/", home), p)

	// missing title should generate error
	p, err = tagTitleToFSDir("", home, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag title required")
	assert.Equal(t, "", p)
}

func TestPathToTagRootTag(t *testing.T) {
	assert.Equal(t, "dotfiles.config.fish", pathToTag(".config/fish/", ""))
	assert.Equal(t, "PersonalDotfiles.config.fish", pathToTag(".config/fish/", "PersonalDotfiles"))
	assert.Equal(t, "WorkDotfiles", pathToTag("", "WorkDotfiles"))
}

func TestResolveRootTag(t *testing.T) {
	got, err := ResolveRootTag("")
	require.NoError(t, err)
	assert.Equal(t, DotFilesTag, got)

	got, err = ResolveRootTag("  PersonalDotfiles  ")
	require.NoError(t, err)
	assert.Equal(t, "PersonalDotfiles", got)

	_, err = ResolveRootTag("Personal.Dotfiles")
	require.ErrorContains(t, err, "must not contain '.'")

	_, err = ResolveRootTag("a/b")
	require.ErrorContains(t, err, "path separators")
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
	newTag, err := createTag("my.test.tag")
	assert.NoError(t, err)
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
	iDiff, err := compareNoteWithFile("apple", applePath, home, appleNote, true)
	require.NoError(t, err)
	assert.Equal(t, identical, iDiff.diff)
	assert.Equal(t, "apple", iDiff.tagTitle)
	assert.Equal(t, "apple", iDiff.noteTitle)
	assert.Equal(t, applePath, iDiff.path)
	assert.Equal(t, appleNote, iDiff.remote)
}

func TestCompareRemoteNewer(t *testing.T) {
	home := getTemporaryHome()
	err := os.MkdirAll(home, os.ModePerm)

	lemonNote := createNote("lemon", "lemon content 2")
	lemonNote.UpdatedAtTimestamp = time.Now().Add(1 * time.Hour).UnixMicro()
	lemonPath := fmt.Sprintf("%s/lemon", home)
	assert.NoError(t, err)

	var f *os.File
	f, err = os.Create(lemonPath)
	assert.NoError(t, err)
	_, err = f.WriteString("lemon content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote differ and remote newer produces correct ItemDiff
	iDiff, cErr := compareNoteWithFile("lemon", lemonPath, home, lemonNote, true)
	require.NoError(t, cErr)
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
	lemonNote.UpdatedAtTimestamp = time.Now().Add(-1 * time.Hour).UnixMicro()
	lemonPath := fmt.Sprintf("%s/lemon", home)
	assert.NoError(t, err)
	var f *os.File
	f, err = os.Create(lemonPath)
	assert.NoError(t, err)
	_, err = f.WriteString("lemon content")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())
	// verify local and remote differ and local newer produces correct ItemDiff
	iDiff, cErr := compareNoteWithFile("lemon", lemonPath, home, lemonNote, true)
	require.NoError(t, cErr)
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
	assert.Equal(t, session.Session{}, sess)

	// ensure an invalid session returns an error, no email address, and an empty session
	email, sess, err = ParseSessionString("someone@example.com;https://sync.standardnotes.org;eyJhbGciOiJKUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c;8f0f5166841ca4dee2975c74cc7e0a4345ce24b54d7b215677a3d540303aa203;6d5ffc6f8e337e6e3ae6d0c3201d9e2d00ffee64672bc4fe1886ad31770c19f1")
	assert.NoError(t, err)
	assert.Equal(t, "someone@example.com", email)
	assert.NotNil(t, sess)
	assert.Equal(t, session.Session{Server: "https://sync.standardnotes.org",
		Token: "eyJhbGciOiJKUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
		sess)
}

func TestNoteWithTagExists(t *testing.T) {
	note := newTestNote()
	nContent := gosn.NewNoteContent()
	nContent.SetTitle("apple")
	note.Content = *nContent
	tContent := gosn.NewTagContent()
	tag := newTestTag()
	tContent.SetTitle("fruit")
	tag.Content = *tContent
	twn := tagsWithNotes{
		tagWithNotes{tag: tag, notes: gosn.Notes{note}},
	}
	assert.Equal(t, 1, noteWithTagExists("fruit", "apple", twn))
}

func TestPushNoItems(t *testing.T) {

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

func TestIsBinaryFile(t *testing.T) {
	dir := t.TempDir()

	// a multi-byte rune straddling the end of the sniff window must not be read
	// as binary just because it was cut in half
	straddling := strings.Repeat("a", binarySniffBytes-1) + "é" + strings.Repeat("a", 100)

	cases := []struct {
		name    string
		content []byte
		binary  bool
	}{
		{name: "plain text", content: []byte("export EDITOR=vim\n"), binary: false},
		{name: "empty", content: []byte{}, binary: false},
		{name: "utf8 text", content: []byte("# ~/.config with héllo and 日本語\n"), binary: false},
		{name: "rune straddling sniff window", content: []byte(straddling), binary: false},
		{name: "nul byte", content: []byte("text\x00more"), binary: true},
		{name: "invalid utf8", content: []byte{0x7f, 'E', 'L', 'F', 0xff, 0xfe}, binary: true},
		// only the start of a file is examined, as git does
		{name: "binary past the sniff window", content: append([]byte(strings.Repeat("a", binarySniffBytes+10)), 0x00), binary: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", dir, strings.ReplaceAll(c.name, " ", "_"))
			require.NoError(t, os.WriteFile(path, c.content, 0o600))

			binary, err := isBinaryFile(path)
			require.NoError(t, err)
			require.Equal(t, c.binary, binary)
		})
	}

	_, err := isBinaryFile(fmt.Sprintf("%s/does-not-exist", dir))
	require.Error(t, err)
}

func TestValidateRootTag(t *testing.T) {
	for _, valid := range []string{"dotfiles", "PersonalDotfiles", "work-dotfiles", "dot_files", "df2"} {
		assert.NoErrorf(t, ValidateRootTag(valid), "expected %q to be a valid root tag", valid)
	}

	// a dot would be read as a path separator in the tag hierarchy
	require.ErrorContains(t, ValidateRootTag("Personal.Dotfiles"), "must not contain '.'")
	require.ErrorContains(t, ValidateRootTag(".dotfiles"), "must not contain '.'")

	require.ErrorContains(t, ValidateRootTag("work/dotfiles"), "path separators")
	require.ErrorContains(t, ValidateRootTag(`work\dotfiles`), "path separators")

	require.ErrorContains(t, ValidateRootTag(""), "must not be empty")
	require.ErrorContains(t, ValidateRootTag("   "), "must not be empty")
}

// rootTagFixture builds a tag with the given title, optionally referencing notes.
func rootTagFixture(t *testing.T, title string, notes ...gosn.Note) *gosn.Tag {
	t.Helper()

	tag := newTestTag()
	content := gosn.NewTagContent()
	content.SetTitle(title)

	var refs gosn.ItemReferences
	for _, note := range notes {
		refs = append(refs, gosn.ItemReference{UUID: note.GetUUID(), ContentType: "Note"})
	}

	if refs != nil {
		content.UpsertReferences(refs)
	}

	tag.Content = *content

	return &tag
}

func noteFixture(t *testing.T, title string) gosn.Note {
	t.Helper()

	note := newTestNote()
	content := gosn.NewNoteContent()
	content.SetTitle(title)
	note.Content = *content

	return note
}

func TestRootTagsFromItems(t *testing.T) {
	gitconfig := noteFixture(t, ".gitconfig")
	shoppingList := noteFixture(t, "shopping list")

	items := gosn.Items{
		&gitconfig,
		&shoppingList,
		// a root: it holds a note whose title starts with a dot
		rootTagFixture(t, "PersonalDotfiles", gitconfig),
		// a root: it has a child tag, even with no notes of its own
		rootTagFixture(t, "WorkDotfiles"),
		rootTagFixture(t, "WorkDotfiles.config"),
		// not a root: an ordinary tag holding an ordinary note
		rootTagFixture(t, "recipes", shoppingList),
		// not a root: dotted titles are children, not roots
		rootTagFixture(t, "PersonalDotfiles.config.fish"),
	}

	roots := rootTagsFromItems(items)

	assert.ElementsMatch(t, []string{"PersonalDotfiles", "WorkDotfiles"}, roots)
}

func TestRootTagsFromItemsEmpty(t *testing.T) {
	assert.Empty(t, rootTagsFromItems(gosn.Items{}))

	// tags with neither dotfile notes nor children are not roots
	plain := noteFixture(t, "notes to self")
	assert.Empty(t, rootTagsFromItems(gosn.Items{&plain, rootTagFixture(t, "misc", plain)}))
}

// editorComponentFixture builds an editor component claiming the given notes.
func editorComponentFixture(t *testing.T, name string, area string, noteUUIDs ...string) *gosn.Component {
	t.Helper()

	component := gosn.NewComponent()

	content := gosn.NewComponentContent()
	content.Name = name
	content.Area = area
	content.AssociateItems(noteUUIDs)

	component.Content = *content

	return &component
}

func TestFindEditorAssociations(t *testing.T) {
	home := "/home/me"

	gitconfig := noteFixture(t, ".gitconfig")
	vimrc := noteFixture(t, ".vimrc")
	unrelated := noteFixture(t, "shopping list")

	dotfilesTag := newTestTag()
	tagContent := gosn.NewTagContent()
	tagContent.SetTitle(DotFilesTag)
	dotfilesTag.Content = *tagContent

	twn := tagsWithNotes{
		tagWithNotes{tag: dotfilesTag, notes: gosn.Notes{gitconfig, vimrc}},
	}

	items := gosn.Items{
		// claims a tracked note: this is what we want reported
		editorComponentFixture(t, "Super", editorArea, gitconfig.GetUUID()),
		// a note that is not tracked, so not our concern
		editorComponentFixture(t, "Rich Text", editorArea, unrelated.GetUUID()),
		// not an editor, so ignored even though it claims a tracked note
		editorComponentFixture(t, "Some Theme", "themes", vimrc.GetUUID()),
	}

	found := findEditorAssociations(items, twn, home, "")

	require.Len(t, found, 1)
	assert.Equal(t, ".gitconfig", found[0].NotePath)
	assert.Equal(t, "Super", found[0].Editor)
}

func TestFindEditorAssociationsNone(t *testing.T) {
	assert.Empty(t, findEditorAssociations(gosn.Items{}, tagsWithNotes{}, "/home/me", ""))
}

func TestEditorAssociationWarning(t *testing.T) {
	assert.Empty(t, editorAssociationWarning(nil))

	msg := editorAssociationWarning([]EditorAssociation{{NotePath: ".gitconfig", Editor: "Super"}})
	assert.Contains(t, msg, ".gitconfig")
	assert.Contains(t, msg, "Super")
	assert.Contains(t, msg, "plain text")
}

// TestCompareNoteWithFileReturnsErrors covers compareNoteWithFile having called
// log.Fatal on a read failure, which exited the process - and, under test, the
// test binary - instead of reporting the problem.
func TestCompareNoteWithFileReturnsErrors(t *testing.T) {
	home := t.TempDir()

	note := noteFixture(t, "apple")

	// a path that does not exist
	_, err := compareNoteWithFile("apple", filepath.Join(home, ".missing"), home, note, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")

	// and one that exists but cannot be read
	unreadable := filepath.Join(home, ".unreadable")
	require.NoError(t, os.WriteFile(unreadable, []byte("secret"), 0o000))

	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o600) })

	if os.Geteuid() == 0 {
		t.Skip("running as root, which can read a 0000 file")
	}

	_, err = compareNoteWithFile("apple", unreadable, home, note, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}
