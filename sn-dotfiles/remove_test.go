package sndotfiles

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/jonhadfield/gosn"

	"github.com/stretchr/testify/assert"
)

func TestRemoveNoItems(t *testing.T) {
	err := remove(gosn.Session{}, gosn.Items{}, true)
	assert.Error(t, err)
}

func TestRemoveItemsInvalidSession(t *testing.T) {
	tag := gosn.NewTag()
	tagContent := gosn.NewTagContent()
	tagContent.SetTitle("newTag")
	err := remove(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, gosn.Items{*tag}, true)
	assert.Error(t, err)
}

func TestRemoveInvalidSession(t *testing.T) {
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))
	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"

	assert.NoError(t, createTemporaryFiles(fwc))
	_, _, _, _, err := Remove(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, home, []string{gitConfigPath}, true)
	assert.Error(t, err)
}

func TestRemoveInvalidPath(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	_, _, _, _, err = Remove(session, getTemporaryHome(), []string{"/invalid"}, true)
	assert.Error(t, err)
}

func TestRemoveNoPaths(t *testing.T) {
	_, _, _, _, err := Remove(gosn.Session{}, getTemporaryHome(), []string{}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "paths")
}

func TestRemoveItems(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))

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
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, applePath, yellowPath, premiumPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai, true)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 4)
	assert.Len(t, ao.PathsExisting, 0)
	assert.Len(t, ao.PathsInvalid, 0)

	// remove single path
	var notesRemoved, tagsRemoved, noNotTracked int
	var msg string
	notesRemoved, tagsRemoved, noNotTracked, msg, err = Remove(session, home, []string{gitConfigPath}, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, notesRemoved)
	assert.Equal(t, 0, tagsRemoved)
	assert.Equal(t, 0, noNotTracked)
	assert.NotEmpty(t, msg)
	re := regexp.MustCompile("\\.gitconfig\\s+removed")
	assert.True(t, re.MatchString(msg))

	// remove nested path with single item (with trailing slash)
	notesRemoved, tagsRemoved, noNotTracked, msg, err = Remove(session, home, []string{fmt.Sprintf("%s/.cars/", home)}, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, notesRemoved)
	assert.Equal(t, 3, tagsRemoved)
	assert.Equal(t, 0, noNotTracked)
	assert.NotEmpty(t, msg)
	re = regexp.MustCompile("\\.cars/mercedes/a250/premium\\s+removed")
	assert.True(t, re.MatchString(msg))

	// remove nested path with single item (without trailing slash)
	notesRemoved, tagsRemoved, noNotTracked, msg, err = Remove(session, home, []string{fmt.Sprintf("%s/.fruit", home)}, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, notesRemoved)
	assert.Equal(t, 3, tagsRemoved)
	assert.Equal(t, 0, noNotTracked)
	assert.NotEmpty(t, msg)
	re = regexp.MustCompile("\\.fruit/apple\\s+removed")
	assert.True(t, re.MatchString(msg))
	re = regexp.MustCompile("\\.fruit/banana/yellow\\s+removed")
	assert.True(t, re.MatchString(msg))

	// ensure error with missing home
	notesRemoved, tagsRemoved, noNotTracked, msg, err = Remove(session, "",
		[]string{fmt.Sprintf("%s/.fruit", home)}, true)
	assert.Error(t, err)

	// ensure error with missing paths
	notesRemoved, tagsRemoved, noNotTracked, msg, err = Remove(session, "home",
		[]string{""}, true)
	assert.Error(t, err)
	fmt.Println(err)
}

func TestRemoveItemsRecursive(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"
	// path to recursively remove
	fruitPath := fmt.Sprintf("%s/.fruit", home)
	// try removing same path twice
	fruitPathDupe := fmt.Sprintf("%s/.fruit", home)

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, applePath, yellowPath, premiumPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai, true)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 4)
	assert.Len(t, ao.PathsExisting, 0)
	var noRemoved, noTagsRemoved, noNotTracked int
	// try removing overlapping path and note in specified path
	noRemoved, noTagsRemoved, noNotTracked, _, err = Remove(session, home, []string{yellowPath, fruitPath, fruitPathDupe}, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, noRemoved)
	assert.Equal(t, 2, noTagsRemoved)
	assert.Equal(t, 0, noNotTracked)
}

func TestRemoveItemsRecursiveTwo(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	greenPath := fmt.Sprintf("%s/.fruit/banana/green", home)
	fwc[greenPath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"
	// path to recursively remove
	//removebananaPath := fmt.Sprintf("%s/.fruit/banana/", Home)
	fruitPath := fmt.Sprintf("%s/.fruit", home)

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, greenPath, yellowPath, premiumPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai, true)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 4)
	assert.Len(t, ao.PathsExisting, 0)
	var noRemoved, noTagsRemoved, noNotTracked int
	noRemoved, noTagsRemoved, noNotTracked, _, err = Remove(session, home, []string{fruitPath}, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, noRemoved)
	assert.Equal(t, 2, noTagsRemoved)
	assert.Equal(t, 0, noNotTracked)
}

func TestRemoveItemsRecursiveThree(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	greenPath := fmt.Sprintf("%s/.fruit/banana/green", home)
	fwc[greenPath] = "apple content"
	yellowPath := fmt.Sprintf("%s/.fruit/banana/yellow", home)
	fwc[yellowPath] = "yellow content"
	premiumPath := fmt.Sprintf("%s/.cars/mercedes/a250/premium", home)
	fwc[premiumPath] = "premium content"
	lokiPath := fmt.Sprintf("%s/.dogs/labrador/loki", home)
	fwc[lokiPath] = "chicken please content"
	// path to recursively remove
	fruitPath := fmt.Sprintf("%s/.fruit/", home)

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, greenPath, yellowPath, premiumPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai, true)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 4)
	assert.Len(t, ao.PathsExisting, 0)
	var noRemoved, noTagsRemoved, noNotTracked int
	noRemoved, noTagsRemoved, noNotTracked, _, err = Remove(session, home, []string{fruitPath, lokiPath}, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, noRemoved)
	assert.Equal(t, 2, noTagsRemoved)
	assert.Equal(t, 1, noNotTracked)
}

func TestRemoveAndCheckRemoved(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git configuration"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai, true)
	assert.NoError(t, err)
	assert.Len(t, ao.PathsAdded, 1)
	assert.Len(t, ao.PathsExisting, 0)
	var noRemoved, noTagsRemoved, noNotTracked int
	noRemoved, noTagsRemoved, noNotTracked, _, err = Remove(session, home, []string{gitConfigPath}, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, noRemoved)
	assert.Equal(t, 1, noTagsRemoved)
	assert.Equal(t, 0, noNotTracked)
	twn, _ := get(session)
	assert.Len(t, twn, 0)
}

func TestRemoveAndCheckRemovedOne(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	defer func() {
		if _, err := WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to WipeTheLot")
		}
	}()
	home := getTemporaryHome()
	debugPrint(true, fmt.Sprintf("test | using temp home: %s", home))

	fwc := make(map[string]string)
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git configuration"
	awsConfigPath := fmt.Sprintf("%s/.aws/config", home)
	fwc[awsConfigPath] = "aws config"
	acmeConfigPath := fmt.Sprintf("%s/.acme/config", home)
	fwc[acmeConfigPath] = "acme config"
	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	// TODO: return notes/files added AND tags added?
	ai := AddInput{Session: session, Home: home, Paths: []string{gitConfigPath, awsConfigPath, acmeConfigPath}, Debug: true}
	var ao AddOutput
	ao, err = Add(ai, true)
	assert.NoError(t, err)
	// dotfiles tag, .gitconfig, and acmeConfig should exist
	assert.Len(t, ao.PathsAdded, 3)
	assert.Len(t, ao.PathsExisting, 0)
	var noRemoved, noTagsRemoved, noNotTracked int
	noRemoved, noTagsRemoved, noNotTracked, _, err = Remove(session, home, []string{gitConfigPath, acmeConfigPath}, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, noRemoved)
	assert.Equal(t, 1, noTagsRemoved)
	assert.Equal(t, 0, noNotTracked)
	twn, _ := get(session)
	// dotfiles tag and .gitconfig note should exist
	assert.Len(t, twn, 2)
}
