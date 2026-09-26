package sndotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gosn "github.com/jonhadfield/gosn-v2/items"
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
		tag: mustCreateTag("something.else.noteOne"),
	},
		tagWithNotes{mustCreateTag("something.else"),
			gosn.Notes{noteOne}},
	}
	err := checkNoteTagConflicts(twn, "")
	assert.Error(t, err)
}

func TestPreflightOverlaps1(t *testing.T) {
	// without overlap
	noteOne := createNote("noteTwo", "hello world")
	twn := tagsWithNotes{tagWithNotes{
		tag: mustCreateTag("something.else.noteOne"),
	},
		tagWithNotes{mustCreateTag("something.else"),
			gosn.Notes{noteOne}},
	}
	err := checkNoteTagConflicts(twn, "")
	assert.NoError(t, err)
}

// TestPreflightResolvesBeforeValidating covers paths being validated as given
// rather than as resolved: a relative path was checked against the working
// directory, so a file that existed under home was reported missing.
func TestPreflightResolvesBeforeValidating(t *testing.T) {
	home := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".config", "fish"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "fish", "config.fish"), []byte("set -x"), 0o600))

	for _, in := range []string{
		".gitconfig",                      // relative to home
		"~/.gitconfig",                    // shell expansion
		".config/fish/config.fish",        // relative, nested
		filepath.Join(home, ".gitconfig"), // already absolute
	} {
		out, err := preflight(home, []string{in})
		require.NoErrorf(t, err, "preflight(%q)", in)
		require.Lenf(t, out, 1, "preflight(%q)", in)
		assert.Truef(t, filepath.IsAbs(out[0]), "preflight(%q) returned %q, which is not absolute", in, out[0])
		assert.Containsf(t, out[0], home, "preflight(%q) returned %q, which is not under home", in, out[0])
	}

	// a path that exists nowhere is still an error
	_, err := preflight(home, []string{".does-not-exist"})
	require.Error(t, err)
}
