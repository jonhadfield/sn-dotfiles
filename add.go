package sndotfiles

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/jonhadfield/gosn"
	"github.com/ryanuber/columnize"
)

// Add tracks local paths by pushing the local dir as a tag representation and the filename as a note title
func Add(session gosn.Session, home string, paths []string, quiet, debug bool) (pathsAdded, pathsExisting, pathsInvalid []string, err error) {
	// remove any duplicate paths
	paths = dedupe(paths)

	if err = checkPathsExist(paths); err != nil {
		return
	}
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	var twn tagsWithNotes
	twn, err = get(session)
	if err != nil {
		return
	}

	err = preflight(twn, paths)
	if err != nil {
		return
	}

	var missing []string
	var existing []string
	var added []string
	tagToItemMap := make(map[string]gosn.Items)

	var finalPaths []string

	// check for directories
	for _, path := range paths {
		// if path is directory, then walk to generate list of additional paths
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					pathsInvalid = append(pathsInvalid, path)
					return fmt.Errorf("failed to read path %q: %v", path, err)
				}
				// ensure walked path is valid
				if !checkPathValid(path) {
					pathsInvalid = append(pathsInvalid, path)
					return nil
				}

				if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
					finalPaths = append(finalPaths, path)
				}
				return nil
			})
		} else {
			finalPaths = append(finalPaths, path)
		}
	}
	if len(finalPaths) == 0 {
		return
	}
	// dedupe
	finalPaths = dedupe(finalPaths)

	for _, path := range finalPaths {
		dir, filename := filepath.Split(path)
		homeRelPath := stripHome(dir+filename, home)
		boldHomeRelPath := bold(homeRelPath)
		// TODO: Remove following as finalPaths must exist
		//if _, err := os.Stat(path); os.IsNotExist(err) {
		//	debugPrint(debug, fmt.Sprintf("add | path does not exist: %s", path))
		//	missing = append(missing, fmt.Sprintf("%s | %s", boldHomeRelPath, red("does not exist")))
		//	pathsInvalid = append(pathsInvalid, path)
		//	continue
		//}
		var remoteTagTitleWithoutHome, remoteTagTitle string
		remoteTagTitleWithoutHome = stripHome(dir, home)
		remoteTagTitle = pathToTag(remoteTagTitleWithoutHome)

		existingCount := noteWithTagExists(remoteTagTitle, filename, twn)
		if existingCount == 1 {
			existing = append(existing, fmt.Sprintf("%s | %s", boldHomeRelPath, yellow("already tracked")))
			pathsExisting = append(pathsExisting, path)
			continue
		} else if existingCount > 1 {
			return pathsAdded, pathsExisting, pathsInvalid,
				fmt.Errorf("duplicate items found with name '%s' and tag '%s'", filename, remoteTagTitle)
		}

		// now add
		pathsAdded = append(pathsAdded, path)

		var itemToAdd gosn.Item
		itemToAdd, err = createItem(path, filename)
		if err != nil {
			return pathsAdded, pathsExisting, pathsInvalid, err
		}
		tagToItemMap[remoteTagTitle] = append(tagToItemMap[remoteTagTitle], itemToAdd)
		added = append(added, fmt.Sprintf("%s | %s", boldHomeRelPath, green("now tracked")))
	}
	lines := append(missing, existing...)

	if err = pushAndTag(session, tagToItemMap, twn); err != nil {
		fmt.Println(columnize.SimpleFormat(lines))
		return pathsAdded, pathsExisting, pathsInvalid, err

	}
	lines = append(lines, added...)
	if !quiet {
		fmt.Println(columnize.SimpleFormat(lines))
	}
	return pathsAdded, pathsExisting, pathsInvalid, err
}

func createItem(path, title string) (item gosn.Item, err error) {
	// read file content
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Println("failed to close file:", path)
		}
	}()
	var localBytes []byte
	localBytes, err = ioutil.ReadAll(file)
	if err != nil {
		return
	}
	localStr := string(localBytes)
	// push item
	item = *gosn.NewNote()
	itemContent := gosn.NewNoteContent()
	item.Content = itemContent
	item.Content.SetTitle(title)
	item.Content.SetText(localStr)
	return
}

func checkPathValid(path string) bool {
	s, err := isSymlink(path)
	if err != nil {
		fmt.Printf("failed to read path: %q %v\n", path, err)
		return false
	}
	if s {
		fmt.Printf("symlinks not currently supported: %q", path)
		return false
	}
	return true
}
