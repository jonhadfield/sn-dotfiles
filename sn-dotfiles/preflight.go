package sndotfiles

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// preflight validates and tidies up the home directory and paths provided
func preflight(home string, in []string) (out []string, err error) {
	// check home is present
	if len(home) == 0 {
		err = errors.New("home undefined")
		return
	}

	// remove any duplicate paths
	in = dedupe(in)

	var v bool

	for _, inPath := range in {
		// handle shell expansion and paths relative to home
		switch {
		case strings.HasPrefix(inPath, "~"):
			inPath = filepath.Join(home, strings.TrimPrefix(inPath, "~"))
		case !filepath.IsAbs(inPath):
			inPath = filepath.Join(home, inPath)
		}

		// validate the resolved path, not the one that was supplied
		if v, err = pathValid(inPath); !v {
			return
		}

		out = append(out, inPath)
	}

	return
}

func checkNoteTagConflicts(twn tagsWithNotes) error {
	// check for path conflict where tag and note overlap
	tagPaths := make(map[string]struct{}, len(twn))
	notePaths := make(map[string]struct{})

	for _, t := range twn {
		tagPath := t.tag.Content.GetTitle()
		tagPaths[tagPath] = struct{}{}

		// loop through tag related notes and generate a list
		// of all combinations to check for duplicates
		for _, n := range t.notes {
			// if tag path is not root (DotFilesTag) then it's a sub tag/dir
			// so add tag path (plus period) to note title, otherwise just add
			// the note title to DotFilesTag
			notePath := tagPath + n.Content.GetTitle()
			if tagPath != DotFilesTag {
				notePath = tagPath + "." + n.Content.GetTitle()
			}

			notePaths[notePath] = struct{}{}
		}
	}

	var overlaps []string

	for notePath := range notePaths {
		if _, found := tagPaths[notePath]; found {
			overlaps = append(overlaps, "- "+notePath)
		}
	}

	if len(overlaps) == 0 {
		return nil
	}

	sort.Strings(overlaps)

	return fmt.Errorf("the following notes and tags are overlapping:\n%s", strings.Join(overlaps, "\n"))
}
