package sndotfiles

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryanuber/columnize"

	"github.com/jonhadfield/gosn"
)

// Add tracks local Paths by pushing the local dir as a tag representation and the filename as a note title
func Add(ai AddInput) (ao AddOutput, err error) {
	// ensure home is passed
	if len(ai.Home) == 0 {
		err = errors.New("home undefined")
		return
	}

	if StringInSlice(ai.Home, []string{"/", "/home"}, true) {
		err = errors.New(fmt.Sprintf("not a good idea to use '%s' as home dir", ai.Home))
	}

	var noRecurse bool
	if ai.All {
		noRecurse = true

		ai.Paths, err = discoverDotfilesInHome(ai.Home, ai.Debug)
		if err != nil {
			return
		}
	}

	// check paths defined
	if len(ai.Paths) == 0 {
		return ao, errors.New("paths not defined")
	}

	// remove any duplicate paths
	ai.Paths = dedupe(ai.Paths)

	// check paths are valid
	if err = checkFSPaths(ai.Paths); err != nil {
		return
	}

	debugPrint(ai.Debug, fmt.Sprintf("Add | paths after dedupe: %d", len(ai.Paths)))

	var twn tagsWithNotes

	twn, err = get(ai.Session)
	if err != nil {
		return
	}

	// run pre-checks
	err = checkNoteTagConflicts(twn)
	if err != nil {
		return
	}

	ai.Twn = twn

	return add(ai, noRecurse, ai.Debug)
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

func add(ai AddInput, noRecurse, debug bool) (ao AddOutput, err error) {
	var tagToItemMap map[string]gosn.Items

	var fsPathsToAdd []string

	// generate list of Paths to add
	fsPathsToAdd, err = getLocalFSPaths(ai.Paths, noRecurse)
	if err != nil {
		return
	}

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

func getLocalFSPaths(paths []string, noRecurse bool) (finalPaths []string, err error) {
	// check for directories
	for _, path := range paths {
		// if path is directory, then walk to generate list of additional Paths
		var stat os.FileInfo
		if stat, err = os.Stat(path); err == nil && stat.IsDir() && !noRecurse {
			err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("failed to read path %q: %v", path, err)
				}
				stat, err = os.Stat(path)
				if err != nil {
					return err
				}
				// if it's a dir, then carry on
				if stat.IsDir() {
					return nil
				}

				// if file is valid, then add
				var valid bool
				valid, err = pathValid(path)
				if err != nil {
					return err
				}
				if valid {
					finalPaths = append(finalPaths, path)
					return err
				}
				return nil
			})
			// return if we failed to walk the dir
			if err != nil {
				return
			}
		} else {
			// path is file
			var valid bool
			valid, err = pathValid(path)
			if err != nil {
				return
			}
			if valid {
				finalPaths = append(finalPaths, path)
			}
		}
	}
	// dedupe
	finalPaths = dedupe(finalPaths)

	return finalPaths, err
}

func createItem(path, title string) (item gosn.Item, err error) {
	// read file content
	var file *os.File

	file, err = os.Open(path)
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

	return item, err
}

func pathInfo(path string) (mode os.FileMode, pathSize int64, err error) {
	var fi os.FileInfo

	fi, err = os.Lstat(path)
	if err != nil {
		return
	}

	mode = fi.Mode()
	if mode.IsRegular() {
		pathSize = fi.Size()
	}

	return
}

func discoverDotfilesInHome(home string, debug bool) (paths []string, err error) {
	debugPrint(debug, fmt.Sprintf("discoverDotfilesInHome | checking home: %s", home))

	var homeEntries []os.FileInfo

	homeEntries, err = ioutil.ReadDir(home)
	if err != nil {
		return
	}

	for _, f := range homeEntries {
		if strings.HasPrefix(f.Name(), ".") {
			var afp string

			afp, err = filepath.Abs(home + string(os.PathSeparator) + f.Name())
			if err != nil {
				return
			}

			if f.Mode().IsRegular() {
				paths = append(paths, afp)
			}
		}
	}

	return
}

func pathValid(path string) (valid bool, err error) {
	var mode os.FileMode

	var pSize int64

	mode, pSize, err = pathInfo(path)
	if err != nil {
		return
	}

	switch {
	case mode.IsRegular():
		if pSize > 10240000 {
			err = fmt.Errorf("file too large: %s", path)
			return false, err
		}

		return true, nil
	case mode&os.ModeSymlink != 0:
		return false, fmt.Errorf("symlink not supported: %s", path)
	case mode.IsDir():
		return true, nil
	case mode&os.ModeSocket != 0:
		return false, fmt.Errorf("sockets not supported: %s", path)
	case mode&os.ModeCharDevice != 0:
		return false, fmt.Errorf("char device file not supported: %s", path)
	case mode&os.ModeDevice != 0:
		return false, fmt.Errorf("device file not supported: %s", path)
	case mode&os.ModeNamedPipe != 0:
		return false, fmt.Errorf("named pipe not supported: %s", path)
	case mode&os.ModeTemporary != 0:
		return false, fmt.Errorf("temporary file not supported: %s", path)
	case mode&os.ModeIrregular != 0:
		return false, fmt.Errorf("irregular file not supported: %s", path)
	default:
		return false, fmt.Errorf("unknown file type: %s", path)
	}
}
