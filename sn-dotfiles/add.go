package sndotfiles

import (
	"fmt"
	"io/ioutil"
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
	bold := color.New(color.Bold).SprintFunc()
	added := make([]string, len(fsPaths))
	var existing []string
	for i, path := range fsPaths {
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
		added[i] = fmt.Sprintf("%s | %s", boldHomeRelPath, green("now tracked"))
	}
	statusLines = append(statusLines, existing...)
	statusLines = append(statusLines, added...)

	return
}

type AddInput struct {
	Session gosn.Session
	Home    string
	Paths   []string
	Debug   bool
}

type AddOutput struct {
	TagsPushed, NotesPushed                 int
	PathsAdded, PathsExisting, PathsInvalid []string
	Msg                                     string
}

// Add tracks local Paths by pushing the local dir as a tag representation and the filename as a note title
func Add(ai AddInput) (ao AddOutput, err error) {
	// remove any duplicate Paths
	ai.Paths = dedupe(ai.Paths)

	if err = checkPathsExist(ai.Paths); err != nil {
		return
	}

	var twn tagsWithNotes
	twn, err = get(ai.Session)
	if err != nil {
		return
	}

	// run pre-checks
	err = preflight(twn, ai.Paths)
	if err != nil {
		return
	}

	var tagToItemMap map[string]gosn.Items
	var fsPathsToAdd []string

	// generate list of Paths to add
	fsPathsToAdd, ao.PathsInvalid = getLocalFSPathsToAdd(ai.Paths)
	if len(fsPathsToAdd) == 0 {
		return
	}

	var statusLines []string
	statusLines, tagToItemMap, ao.PathsAdded, ao.PathsExisting, err = generateTagItemMap(fsPathsToAdd, ai.Home, twn)
	if err != nil {
		return
	}

	// add DotFilesTag tag if missing
	_, dotFilesTagInTagToItemMap := tagToItemMap[DotFilesTag]
	if !tagExists("dotfiles", twn) && !dotFilesTagInTagToItemMap {
		debugPrint(ai.Debug, "Add | adding missing dotfiles tag")
		tagToItemMap[DotFilesTag] = gosn.Items{}
	}

	// push and tag items
	ao.TagsPushed, ao.NotesPushed, err = pushAndTag(ai.Session, tagToItemMap, twn)
	if err != nil {
		return
	}
	ao.Msg = fmt.Sprint(columnize.SimpleFormat(statusLines))

	return ao, err
}

func getLocalFSPathsToAdd(paths []string) (finalPaths, pathsInvalid []string) {
	// check for directories
	for _, path := range paths {
		// if path is directory, then walk to generate list of additional Paths
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
		return
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
