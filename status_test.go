package sndotfiles

import (
	"fmt"
	"os"
	"testing"

	"github.com/lithammer/shortuuid"
	"github.com/stretchr/testify/assert"
)

func TestStatus(t *testing.T) {
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
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{gitConfigPath, applePath, yellowPath, premiumPath}, true, true)
	assert.NoError(t, err)
	assert.Len(t, added, 4)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
	var diffs []ItemDiff

	diffs, err = Status(session, home, []string{gitConfigPath, applePath, yellowPath, premiumPath}, true, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 4)
	var pDiff int
	for _, d := range diffs {
		switch d.path {
		case gitConfigPath:
			assert.Equal(t, ".gitconfig", d.noteTitle)
			assert.Equal(t, gitConfigPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case applePath:
			assert.Equal(t, "apple", d.noteTitle)
			assert.Equal(t, applePath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case yellowPath:
			assert.Equal(t, "yellow", d.noteTitle)
			assert.Equal(t, yellowPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case premiumPath:
			assert.Equal(t, "premium", d.noteTitle)
			assert.Equal(t, premiumPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		}
	}
	assert.Equal(t, 4, pDiff)
}

func TestStatus1(t *testing.T) {
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
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	fwc[gitConfigPath] = "git config content"
	awsConfig := fmt.Sprintf("%s/.aws/config", home)
	fwc[awsConfig] = "aws config content"

	assert.NoError(t, createTemporaryFiles(fwc))
	// add items
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{gitConfigPath, awsConfig}, true, true)
	assert.NoError(t, err)
	assert.Len(t, added, 2)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
	var diffs []ItemDiff

	diffs, err = Status(session, home, []string{gitConfigPath}, true, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 1)
	assert.Equal(t, ".gitconfig", diffs[0].noteTitle)
	assert.Equal(t, gitConfigPath, diffs[0].path)
	assert.Equal(t, identical, diffs[0].diff)
}

func TestStatus2(t *testing.T) {
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
	var added, existing, missing []string
	added, existing, missing, err = Add(session, home, []string{gitConfigPath, applePath, yellowPath, premiumPath}, true, true)
	assert.NoError(t, err)
	assert.Len(t, added, 4)
	assert.Len(t, existing, 0)
	assert.Len(t, missing, 0)
	var diffs []ItemDiff

	diffs, err = Status(session, home, []string{fmt.Sprintf("%s/.fruit", home)}, true, true)
	assert.NoError(t, err)
	assert.Len(t, diffs, 2)
	var pDiff int
	for _, d := range diffs {
		switch d.path {
		case gitConfigPath:
			assert.Equal(t, ".gitconfig", d.noteTitle)
			assert.Equal(t, gitConfigPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case applePath:
			assert.Equal(t, "apple", d.noteTitle)
			assert.Equal(t, applePath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case yellowPath:
			assert.Equal(t, "yellow", d.noteTitle)
			assert.Equal(t, yellowPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		case premiumPath:
			assert.Equal(t, "premium", d.noteTitle)
			assert.Equal(t, premiumPath, d.path)
			assert.Equal(t, identical, d.diff)
			pDiff++
		}
	}
	assert.Equal(t, 2, pDiff)
}
