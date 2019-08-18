package sndotfiles

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/jonhadfield/gosn"
	"github.com/ryanuber/columnize"
)

// Remove stops tracking local paths by removing the related notes from SN
func Remove(session gosn.Session, home string, paths []string, quiet, debug bool) (notesremoved, tagsRemoved, notTracked int, err error) {
	// remove any duplicate paths
	paths = dedupe(paths)

	// verify paths before delete
	if err = checkPathsExist(paths); err != nil {
		return
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	tagsWithNotes, err := get(session)
	if err != nil {
		return
	}

	err = preflight(tagsWithNotes, paths)
	if err != nil {
		return
	}

	var results []string
	var notesToRemove gosn.Items
	for _, path := range paths {

		homeRelPath, matchingItems := getItemsToRemove(path, home, tagsWithNotes)
		boldHomeRelPath := bold(stripTrailingSlash(homeRelPath))
		debugPrint(debug, fmt.Sprintf("Remove | items matching path '%s': %d", path, len(matchingItems)))
		switch {
		case len(matchingItems) == 0:
			results = append(results, fmt.Sprintf("%s | %s", boldHomeRelPath, yellow("not tracked")))
			notTracked++
		case len(matchingItems) == 1:
			results = append(results, fmt.Sprintf("%s | %s", boldHomeRelPath, green("removed")))
			notesToRemove = append(notesToRemove, matchingItems...)
		case len(matchingItems) > 1:
			// TODO: consider displaying additional items to user, rather than just a number
			results = append(results, fmt.Sprintf("%s (%d instances) | %s", boldHomeRelPath, len(matchingItems), green("removed")))
			notesToRemove = append(notesToRemove, matchingItems...)
		}
	}

	// dedupe any notes to remove
	if notesToRemove != nil {
		notesToRemove.DeDupe()
	}

	// find any empty tags to delete
	emptyTags := findEmptyTags(tagsWithNotes, notesToRemove, debug)

	// dedupe any tags to remove
	if emptyTags != nil {
		emptyTags.DeDupe()
	}

	// add empty tags to list of items to remove
	itemsToRemove := append(notesToRemove, emptyTags...)

	debugPrint(debug, fmt.Sprintf("Remove | items to remove: %d", len(itemsToRemove)))

	if err = remove(session, itemsToRemove, debug); err != nil {
		return
	}
	if !quiet {
		fmt.Println(columnize.SimpleFormat(results))
	}
	return len(notesToRemove), len(emptyTags), notTracked, err
}

func stripTrailingSlash(in string) string {
	if strings.HasSuffix(in, "/") {
		return in[:len(in)-1]
	}
	return in
}

func remove(session gosn.Session, items gosn.Items, debug bool) (err error) {
	var itemsToRemove gosn.Items
	for _, item := range items {
		item.Deleted = true
		itemsToRemove = append(itemsToRemove, item)
	}
	if itemsToRemove == nil {
		return fmt.Errorf("no items to remove")
	}
	var pio gosn.PutItemsOutput
	pio, err = putItems(session, itemsToRemove)
	debugPrint(debug, fmt.Sprintf("remove | items put: %d", len(pio.ResponseBody.SavedItems)))
	return err
}
