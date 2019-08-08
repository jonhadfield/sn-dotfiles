package sndotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
)

func TestPathIsPrefixOfPaths(t *testing.T) {
	// path without dir
	assert.False(t, pathIsPrefixOfPaths("myPath", []string{"notMyPath"}))
	// path match on second
	assert.True(t, pathIsPrefixOfPaths("/tmp/apple/", []string{"/tmp/lemon/test.txt", "/tmp/apple/test.txt"}))
	// path with incorrect case
	assert.False(t, pathIsPrefixOfPaths("/tmp/Apple/", []string{"/tmp/lemon/test.txt", "/tmp/apple/test.txt"}))
	// missing path
	assert.False(t, pathIsPrefixOfPaths("", []string{"/tmp/lemon/test.txt", "/tmp/apple/test.txt"}))
	// missing paths
	assert.False(t, pathIsPrefixOfPaths("/tmp/apple/", []string{}))
}

func TestNoteInPaths(t *testing.T) {
	// note but no paths
	assert.False(t, noteInPaths("/tmp/myNote.txt", []string{}))
	// note but empty path
	assert.False(t, noteInPaths("/tmp/myNote.txt", []string{""}))
	// note not in path
	assert.False(t, noteInPaths("/tmp/myNote.txt", []string{"/tmp/myNote.doc", "/tmp/lemon.txt"}))
	// note in second path
	assert.True(t, noteInPaths("/tmp/myNote.txt", []string{"/tmp/myNote.doc", "/tmp/myNote.txt"}))
}

func testDiffSetup1and2(home string) (twn tagsWithNotes, fwc map[string]string) {
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

func TestDiff1(t *testing.T) {
	home := getTemporaryHome()
	twn, filesWithContent := testDiffSetup1and2(home)
	var diffs []ItemDiff
	err := createTemporaryFiles(filesWithContent)
	assert.NoError(t, err)
	defer func() {
		if err := deleteTemporaryFiles(home); err != nil {
			fmt.Printf("failed to clean-up: %s\ndetails: %v\n", home, err)
		}
	}()

	// missing remote and missing local
	_, err = diff(tagsWithNotes{}, home, []string{"missing-file"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tags with notes not supplied")

	// existing remote and missing local
	_, err = diff(twn, home, []string{"missing-file"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")

	// valid local, valid remote, grape not diff'd as not specified in path
	applePath := fmt.Sprintf("%s/.sn-dotfiles-test-fruit/apple", home)
	lemonPath := fmt.Sprintf("%s/.sn-dotfiles-test-fruit/lemon", home)
	allPaths := []string{applePath, lemonPath}
	diffs, err = diff(twn, home, allPaths)
	assert.NoError(t, err)
	assert.Len(t, diffs, 2)
	assert.NotEmpty(t, diffs)

	var foundCount int
	for _, diff := range diffs {
		if diff.noteTitle == "apple" {
			foundCount++
			assert.Equal(t, diff.diff, identical)
			assert.Equal(t, diff.path, applePath)
			assert.Equal(t, "apple content", diff.remote.Content.GetText())
			assert.Equal(t, "apple content", diff.local)

		}
		if diff.noteTitle == "lemon" {
			foundCount++
			assert.Equal(t, diff.diff, identical)
			assert.Equal(t, diff.path, lemonPath)
			assert.Equal(t, "lemon content", diff.remote.Content.GetText())
			assert.Equal(t, "lemon content", diff.local)
		}

		// would not expect to find grape as we supplied specific paths to check
		if diff.noteTitle == "grape" {
			foundCount++
		}
	}
	assert.Equal(t, 2, foundCount)
}

// func testDiffSetup1and2(home string) (twn tagsWithNotes, fwc map[string]string) {
//	fruitTag := createTag("dotfiles.sn-dotfiles-test-fruit")
//	appleNote := createNote("apple", "apple content")
//	lemonNote := createNote("lemon", "lemon content")
//	grapeNote := createNote("grape", "grape content")
//	fruitTagWithNotes := tagWithNotes{tag: fruitTag, notes: gosn.Items{appleNote, lemonNote, grapeNote}}
//	twn = tagsWithNotes{fruitTagWithNotes}
//
//	fwc = make(map[string]string)
//	fwc[fmt.Sprintf("%s/.sn-dotfiles-test-fruit/apple", home)] = "apple content"
//	fwc[fmt.Sprintf("%s/.sn-dotfiles-test-fruit/lemon", home)] = "lemon content"
//	return
//
//}
func TestDiff2(t *testing.T) {
	home := getTemporaryHome()
	twn, filesWithContent := testDiffSetup1and2(home)
	var diffs []ItemDiff
	err := createTemporaryFiles(filesWithContent)
	assert.NoError(t, err)
	defer func() {
		if err := deleteTemporaryFiles(home); err != nil {
			fmt.Printf("failed to clean-up: %s\ndetails: %v\n", home, err)
		}
	}()

	// valid local, valid remote, grape not diff'd as not specified in path
	paths := []string{fmt.Sprintf("%s/.sn-dotfiles-test-fruit/", home)}
	diffs, err = diff(twn, home, paths)
	assert.NoError(t, err)
	assert.Len(t, diffs, 3)
	assert.NotEmpty(t, diffs)
	var foundCount int
	for _, diff := range diffs {
		if diff.noteTitle == "apple" {
			foundCount++
			assert.Equal(t, diff.diff, identical)
			assert.Equal(t, "apple content", diff.remote.Content.GetText())
			assert.Equal(t, "apple content", diff.local)

		}
		if diff.noteTitle == "lemon" {
			foundCount++
			assert.Equal(t, diff.diff, identical)
			assert.Equal(t, "lemon content", diff.remote.Content.GetText())
			assert.Equal(t, "lemon content", diff.local)
		}

		if diff.noteTitle == "grape" {
			assert.Equal(t, "grape content", diff.remote.Content.GetText())
			assert.Empty(t, diff.local)
			foundCount++
		}
	}
	assert.Equal(t, 3, foundCount)
}

func TestDiff3(t *testing.T) {
	home := getTemporaryHome()
	fruitTag := createTag("dotfiles")
	appleNote := createNote(".apple", "apple content")
	fruitTagWithNotes := tagWithNotes{tag: fruitTag, notes: gosn.Items{appleNote}}
	twn := tagsWithNotes{fruitTagWithNotes}

	fwc := make(map[string]string)
	fwc[fmt.Sprintf("%s/.apple", home)] = "apple content"

	var diffs []ItemDiff
	err := createTemporaryFiles(fwc)
	assert.NoError(t, err)
	defer func() {
		if err := deleteTemporaryFiles(home); err != nil {
			fmt.Printf("failed to clean-up: %s\ndetails: %v\n", home, err)
		}
	}()

	// valid local, valid remote, grape not diff'd as not specified in path
	paths := []string{fmt.Sprintf("%s/.apple", home)}
	diffs, err = diff(twn, home, paths)
	assert.NoError(t, err)
	assert.Len(t, diffs, 1)
	assert.Equal(t, identical, diffs[0].diff)
	assert.NotEmpty(t, diffs)

	var foundCount int
	for _, diff := range diffs {
		if diff.noteTitle == ".apple" {
			foundCount++
			assert.Equal(t, diff.diff, identical)
			assert.Equal(t, "apple content", diff.remote.Content.GetText())
			assert.Equal(t, "apple content", diff.local)

		}
	}
}

// helpers
func createNote(title, content string) gosn.Item {
	noteContent := gosn.NewNoteContent()
	noteContent.Title = title
	noteContent.Text = content
	note := gosn.NewNote()
	note.Content = noteContent
	note.ContentType = "Note"
	return *note
}

func createTemporaryFiles(fwc map[string]string) error {
	for f, c := range fwc {
		if err := createPathWithContent(f, c); err != nil {
			return err
		}
	}
	return nil
}

func deleteTemporaryFiles(path string) error {
	// check path is child of temp directory
	if !strings.HasPrefix(path, os.TempDir()) {
		return fmt.Errorf("path: %s is not a child of %s", os.TempDir(), path)
	}
	return os.RemoveAll(path)
}

func createPathWithContent(path, content string) error {
	dir, _ := filepath.Split(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return f.Close()
}
