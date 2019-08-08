package sndotfiles

import (
	"fmt"
	"strings"

	"github.com/fatih/set"
)

func preflight(twn tagsWithNotes) error {
	// check for path conflict where tag and note overlap
	tagPaths := set.New(set.NonThreadSafe)
	notePaths := set.New(set.NonThreadSafe)
	for _, t := range twn {
		tagPath := t.tag.Content.GetTitle()
		tagPaths.Add(tagPath)
		for _, n := range t.notes {
			var notePath string
			if tagPath != DotFilesTag {
				notePath = tagPath + "." + n.Content.GetTitle()
			} else {
				notePath = tagPath + n.Content.GetTitle()

			}
			notePaths.Add(notePath)
		}
	}
	inter := set.Intersection(tagPaths, notePaths)
	var overlaps []string
	for _, i := range inter.List() {
		overlaps = append(overlaps, "- "+i.(string))
	}
	if inter.IsEmpty() {
		return nil
	}
	return fmt.Errorf("the following notes and tags are overlapping:\n%s", strings.Join(overlaps, "\n"))
}
