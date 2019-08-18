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

func generateTagItemMap(fsPaths []string, home string, twn tagsWithNotes) (statusLines []string, tagToItemMap map[string]gosn.Items, pathsAdded, pathsExisting []string, err error) {
	tagToItemMap = make(map[string]gosn.Items)
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	//red := color.New(color.FgRed).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	var added, existing, missing []string
	for _, path := range fsPaths {
		dir, filename := filepath.Split(path)
		homeRelPath := stripHome(dir+filename, home)
		boldHomeRelPath := bold(homeRelPath)

		var remoteTagTitleWithoutHome, remoteTagTitle string
		remoteTagTitleWithoutHome = stripHome(dir, home)
		remoteTagTitle = pathToTag(remoteTagTitleWithoutHome)

		existingCount := noteWithTagExists(remoteTagTitle, filename, twn)
		if existingCount == 1 {
			existing = append(existing, fmt.Sprintf("%s | %s", boldHomeRelPath, yellow("already tracked")))
			pathsExisting = append(pathsExisting, path)
			continue
		} else if existingCount > 1 {
			err = fmt.Errorf("duplicate items found with name '%s' and tag '%s'", filename, remoteTagTitle)
			return
		}
		// now add
		pathsAdded = append(pathsAdded, path)

		var itemToAdd gosn.Item
		itemToAdd, err = createItem(path, filename)
		if err != nil {
			return
		}
		tagToItemMap[remoteTagTitle] = append(tagToItemMap[remoteTagTitle], itemToAdd)
		added = append(added, fmt.Sprintf("%s | %s", boldHomeRelPath, green("now tracked")))
	}
	statusLines = append(missing, existing...)
	statusLines = append(statusLines, added...)

	return
}

// Add tracks local paths by pushing the local dir as a tag representation and the filename as a note title
func Add(session gosn.Session, home string, paths []string, quiet, debug bool) (pathsAdded, pathsExisting, pathsInvalid []string, err error) {
	// remove any duplicate paths
	paths = dedupe(paths)

	if err = checkPathsExist(paths); err != nil {
		return
	}

	var twn tagsWithNotes
	twn, err = get(session)
	if err != nil {
		return
	}

	// run pre-checks
	err = preflight(twn, paths)
	if err != nil {
		return
	}

	var tagToItemMap map[string]gosn.Items
	var fsPathsToAdd []string

	// generate list of paths to add
	fsPathsToAdd, pathsInvalid = getLocalFSPathsToAdd(paths)
	if len(fsPathsToAdd) == 0 {
		return
	}

	var statusLines []string
	statusLines, tagToItemMap, pathsAdded, pathsExisting, err = generateTagItemMap(fsPathsToAdd, home, twn)
	if err != nil {
		return
	}

	// add DotFilesTag tag if missing
	_, dotFilesTagInTagToItemMap := tagToItemMap[DotFilesTag]
	if !tagExists("dotfiles", twn) && !dotFilesTagInTagToItemMap {
		debugPrint(debug, "Add | adding missing dotfiles tag")
		tagToItemMap[DotFilesTag] = gosn.Items{}
	}

	// push and tag items
	_, err = pushAndTag(session, tagToItemMap, twn)
	if err != nil {
		return
	}
	if !quiet {
		fmt.Println(columnize.SimpleFormat(statusLines))
	}
	return pathsAdded, pathsExisting, pathsInvalid, err
}

func getLocalFSPathsToAdd(paths []string) (finalPaths, pathsInvalid []string) {
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
	// dedupe
	finalPaths = dedupe(finalPaths)
	return
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
