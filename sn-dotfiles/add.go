package sndotfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryanuber/columnize"

	"github.com/jonhadfield/gosn"
)

func discoverDotfiles(home string) (paths []string, err error) {
	var homeEntries []os.FileInfo

	homeEntries, err = ioutil.ReadDir(home)
	if err != nil {
		return
	}

	for _, f := range homeEntries {
		if strings.HasPrefix(f.Name(), ".") {
			var afp string
			afp, err = filepath.Abs(home + string(os.PathSeparator) + f.Name())
			paths = append(paths, afp)
		}
	}

	return
}

// Add tracks local Paths by pushing the local dir as a tag representation and the filename as a note title
func Add(ai AddInput, debug bool) (ao AddOutput, err error) {
	if ai.All {
		ai.Paths, err = discoverDotfiles(ai.Home)
	}

	// remove any duplicate Paths
	ai.Paths = dedupe(ai.Paths)

	if err = checkPathsExist(ai.Paths); err != nil {
		return
	}

	debugPrint(debug, fmt.Sprintf("Add | paths after dedupe: %d", len(ai.Paths)))

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

	ai.Twn = twn

	return add(ai, debug)
}

func add(ai AddInput, debug bool) (ao AddOutput, err error) {
	var tagToItemMap map[string]gosn.Items

	var fsPathsToAdd []string

	// generate list of Paths to add
	fsPathsToAdd, ao.PathsInvalid = getLocalFSPathsToAdd(ai.Paths)
	if len(fsPathsToAdd) == 0 {
		return
	}

	var statusLines []string

	statusLines, tagToItemMap, ao.PathsAdded, ao.PathsExisting, err = generateTagItemMap(fsPathsToAdd, ai.Home, ai.Twn)
	if err != nil {
		return
	}

	// add DotFilesTag tag if missing
	_, dotFilesTagInTagToItemMap := tagToItemMap[DotFilesTag]
	if !tagExists("dotfiles", ai.Twn) && !dotFilesTagInTagToItemMap {
		debugPrint(ai.Debug, "Add | adding missing dotfiles tag")

		tagToItemMap[DotFilesTag] = gosn.Items{}
	}

	// push and tag items
	ao.TagsPushed, ao.NotesPushed, err = pushAndTag(ai.Session, tagToItemMap, ai.Twn)

	if err != nil {
		return
	}

	debugPrint(debug, fmt.Sprintf("Add | tags pushed: %d notes pushed %d", ao.TagsPushed, ao.NotesPushed))

	ao.Msg = fmt.Sprint(columnize.SimpleFormat(statusLines))

	return ao, err
}

type AddInput struct {
	Session gosn.Session
	Home    string
	Paths   []string
	All     bool
	Debug   bool
	Twn     tagsWithNotes
}

type AddOutput struct {
	TagsPushed, NotesPushed                 int
	PathsAdded, PathsExisting, PathsInvalid []string
	Msg                                     string
}

func generateTagItemMap(fsPaths []string, home string, twn tagsWithNotes) (statusLines []string,
	tagToItemMap map[string]gosn.Items, pathsAdded, pathsExisting []string, err error) {
	tagToItemMap = make(map[string]gosn.Items)

	var added []string

	var existing []string

	for _, path := range fsPaths {
		dir, filename := filepath.Split(path)
		homeRelPath := stripHome(dir+filename, home)
		boldHomeRelPath := bold(homeRelPath)

		var remoteTagTitleWithoutHome, remoteTagTitle string
		remoteTagTitleWithoutHome = stripHome(dir, home)
		remoteTagTitle = pathToTag(remoteTagTitleWithoutHome)

		existingCount := noteWithTagExists(remoteTagTitle, filename, twn)
		if existingCount > 0 {
			existing = append(existing, fmt.Sprintf("%s | %s", boldHomeRelPath, yellow("already tracked")))
			pathsExisting = append(pathsExisting, path)

			continue
		} else if existingCount > 1 {
			err = fmt.Errorf("duplicate items found with name '%s' and tag '%s'", filename, remoteTagTitle)
			return statusLines, tagToItemMap, pathsAdded, pathsExisting, err
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

	statusLines = append(statusLines, existing...)
	statusLines = append(statusLines, added...)

	return statusLines, tagToItemMap, pathsAdded, pathsExisting, err
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
