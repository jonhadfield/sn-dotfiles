package sndotfiles

import (
	"fmt"
	"testing"

	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
)

func TestPreflightInvalidPaths(t *testing.T) {
	home := getTemporaryHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.apple", home)
	fwc[applePath] = "apple content"
	duffPath := fmt.Sprintf("%s/.invalid/dodgy", home)
	twn := tagsWithNotes{}
	assert.Error(t, preflight(twn, []string{applePath, duffPath}))

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
	err := preflight(twn, []string{})
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
	err := preflight(twn, []string{})
	assert.NoError(t, err)
}
