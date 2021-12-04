package sndotfiles

import (
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"testing"

	"github.com/stretchr/testify/assert"
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
			gosn.Notes{noteOne}},
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
			gosn.Notes{noteOne}},
	}
	err := checkNoteTagConflicts(twn)
	assert.NoError(t, err)
}
