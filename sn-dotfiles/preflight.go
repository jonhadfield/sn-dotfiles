package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/fatih/set"
	"path/filepath"
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

	// handle shell expansion
	var v bool
	for _, inPath := range in {
		// handle shell expansion and paths relative to home
		switch {
		case strings.HasPrefix(inPath, "~"):
			inPath = filepath.Join(home, strings.TrimPrefix(inPath, "~"))
		case !filepath.IsAbs(inPath):
			inPath = filepath.Join(home, inPath)
		}

		// validate the resolved path rather than the one supplied: a relative
		// path was being checked against the working directory, so a file that
		// existed under home was reported as missing
		if v, err = pathValid(inPath); !v {
			return
		}

		out = append(out, inPath)
	}

	return
}

func checkNoteTagConflicts(twn tagsWithNotes, rootTag string) error {
	rootTag = normalizeRootTag(rootTag)

	// check for path conflict where tag and note overlap
	tagPaths := set.New(set.NonThreadSafe)
	notePaths := set.New(set.NonThreadSafe)

	for _, t := range twn {
		tagPath := t.tag.Content.GetTitle()
		tagPaths.Add(tagPath)
		// loop through tag related notes and generate a list
		// of all combinations to check for duplicates
		for _, n := range t.notes {
			var notePath string
			// if tag path is not root then it's a sub tag/dir
			// so add tag path (plus period) to note title
			if tagPath != rootTag {
				notePath = tagPath + "." + n.Content.GetTitle()
			} else {
				// otherwise, just add note title to root tag (note titles at home start with '.')
				notePath = tagPath + n.Content.GetTitle()
			}

			notePaths.Add(notePath)
		}
	}

	inter := set.Intersection(tagPaths, notePaths)
	overlaps := make([]string, len(inter.List()))

	for c, i := range inter.List() {
		overlaps[c] = "- " + i.(string)
	}

	if inter.IsEmpty() {
		return nil
	}

	return fmt.Errorf("the following notes and tags are overlapping:\n%s", strings.Join(overlaps, "\n"))
}
