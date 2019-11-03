package sndotfiles

import (
	"fmt"
	"testing"

	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
)

func TestPreflightInvalidPaths(t *testing.T) {
	home := getTemporaryHome()
	duffPath := fmt.Sprintf("%s/.invalid/dodgy", home)
	assert.Error(t, checkFSPaths([]string{duffPath}))

}
func TestPreflightOverlaps(t *testing.T) {
	// with overlap
	noteOne := createNote("noteOne", "hello world")
	twn := tagsWithNotes{tagWithNotes{
		tag: createTag("something.else.noteOne"),
	},
		tagWithNotes{createTag("something.else"),
			gosn.Items{noteOne}},
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
			gosn.Items{noteOne}},
	}
	err := checkNoteTagConflicts(twn)
	assert.NoError(t, err)
}
