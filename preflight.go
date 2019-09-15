package sndotfiles

import (
	"fmt"
	"strings"

	"github.com/fatih/set"
)

func preflight(twn tagsWithNotes, paths []string) error {
	// check all provided Paths are valid
	for i := range paths {
		if !checkPathValid(paths[i]) {
			return fmt.Errorf("path \"%s\" does not exist", paths[i])
		}
	}

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
			// if tag path is not root (DotFilesTag) then it's a sub tag/dir
			// so add tag path (plus period) to note title
			if tagPath != DotFilesTag {
				notePath = tagPath + "." + n.Content.GetTitle()
			} else {
				// otherwise, just add note title to DotFilesTag
				notePath = tagPath + n.Content.GetTitle()
			}
			notePaths.Add(notePath)
		}
	}
	inter := set.Intersection(tagPaths, notePaths)
	overlaps := make([]string, len(inter.List()))
	for _, i := range inter.List() {
		overlaps = append(overlaps, "- "+i.(string))
	}
	if inter.IsEmpty() {
		return nil
	}
	return fmt.Errorf("the following notes and tags are overlapping:\n%s", strings.Join(overlaps, "\n"))
}
