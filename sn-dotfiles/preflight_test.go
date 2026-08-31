package sndotfiles

import (
	"fmt"
	"testing"

	"github.com/jonhadfield/gosn-v2/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreflightInvalidPaths(t *testing.T) {
	home := getTemporaryHome()
	duffPath := fmt.Sprintf("%s/.invalid/dodgy", home)

	_, err := preflight(home, []string{duffPath})
	assert.Error(t, err)
}
func TestPreflightOverlaps(t *testing.T) {
	// with overlap
	noteOne := createNote("noteOne", "hello world")
	twn := tagsWithNotes{tagWithNotes{
		tag: createTag("something.else.noteOne"),
	},
		tagWithNotes{createTag("something.else"),
			items.Notes{noteOne}},
	}
	err := checkNoteTagConflicts(twn)
	assert.Error(t, err)
}

func TestPreflightOverlaps1(t *testing.T) {
	// without overlap
	noteOne := createNote("noteTwo", "hello world")
	twn := tagsWithNotes{tagWithNotes{
		tag: createTag("something.else.noteOne"),
	},
		tagWithNotes{createTag("something.else"),
			items.Notes{noteOne}},
	}
	err := checkNoteTagConflicts(twn)
	assert.NoError(t, err)
}

// TestPreflightResolvesPathsAgainstHome covers preflight having validated the
// path as supplied rather than the one it resolved, which meant a relative path
// was checked against the process working directory instead of home.
func TestPreflightResolvesPathsAgainstHome(t *testing.T) {
	home := getTemporaryHome()
	nvimConfig := fmt.Sprintf("%s/.config/nvim/init.vim", home)

	require.NoError(t, createTemporaryFiles(map[string]string{nvimConfig: "set nocompatible"}))

	for _, in := range []string{
		".config/nvim/init.vim",   // relative to home
		"~/.config/nvim/init.vim", // shell style
		nvimConfig,                // already absolute
	} {
		out, err := preflight(home, []string{in})
		require.NoError(t, err, in)
		assert.Equal(t, []string{nvimConfig}, out, in)
	}
}
