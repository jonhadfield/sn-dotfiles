package sndotfiles

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonhadfield/gosn"
)

const (
	localMissing = "local missing"
	localNewer   = "local newer"
	remoteNewer  = "remote newer"
	untracked    = "untracked"
	identical    = "identical"
)

func pathIsPrefixOfPaths(path string, paths []string) bool {
	for i := range paths {
		inSliceDIR, _ := filepath.Split(paths[i])
		if inSliceDIR == "" {
			continue
		}
		if path == inSliceDIR || strings.HasPrefix(path, inSliceDIR) {
			return true
		}
	}
	return false
}

func noteInPaths(note string, paths []string) bool {
	if note == "" || len(paths) == 0 {
		return false
	}
	for i := range paths {
		if paths[i] == "" {
			continue
		}
		if note == paths[i] {
			return true
		}
		d, _ := filepath.Split(note)
		if d == paths[i] {
			return true
		}
	}
	return false
}

func checkPathsExist(paths []string) error {
	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func diff(remote tagsWithNotes, home string, paths []string) (diffs []ItemDiff, err error) {
	// fail immediately if remote or paths are empty
	if len(remote) == 0 {
		return nil, fmt.Errorf("tags with notes not supplied")
	}
	// check paths specified
	if len(paths) > 0 {
		if err := checkPathsExist(paths); err != nil {
			return nil, err
		}
	}

	var itemDiffs []ItemDiff
	var trackedPaths []string
	// loop through remotes, looking for match for paths
	for _, twn := range remote {
		// only diff if path equals translated tag
		tagTitle := twn.tag.Content.GetTitle()
		var dir string
		dir, _, err = tagTitleToFSDIR(twn.tag.Content.GetTitle(), home)
		if err != nil {
			return
		}

		// check the tag (dir) is equal to, or a prefix of any tag being checked
		if len(paths) > 0 {
			if !pathIsPrefixOfPaths(dir, paths) {
				continue
			}
		}

		// loop through notes and compare content of any with matching file
		// log each matching path so we can later walk them to discover untracked files
		for _, d := range twn.notes {
			fullPath := fmt.Sprintf("%s%s", dir, d.Content.GetTitle())
			// skip if note if exact path is not specified and does not have prefix of total path
			if len(paths) > 0 && !noteInPaths(dir+d.Content.GetTitle(), paths) {
				continue
			}

			if !localExists(fullPath) {
				trackedPaths = append(trackedPaths, fullPath)
				var homeRelPath string
				homeRelPath, err = stripHome(fullPath, home)
				if err != nil {
					return
				}
				itemDiffs = append(itemDiffs, ItemDiff{
					tagTitle:    tagTitle,
					homeRelPath: homeRelPath,
					path:        fullPath,
					diff:        localMissing,
					noteTitle:   d.Content.GetTitle(),
					remote:      d,
				})
			} else {
				trackedPaths = append(trackedPaths, fullPath)
				itemDiffs = append(itemDiffs, compare(tagTitle, fullPath, home, d))
			}
		}
	}
	// add diffs for untracked by comparing specified paths with those found
	if len(paths) > 0 {
		// if path is directory, then walk to generate list of additional paths
		for _, path := range paths {
			_, f := filepath.Split(path)
			if f != "" {
				continue
			}
			if stringInSlice(path, trackedPaths, true) {
				continue
			}
			if stat, err := os.Stat(path); err == nil && stat.IsDir() {
				err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
					// don't walk tracked paths
					if stringInSlice(p, trackedPaths, true) {
						return nil
					}
					if err != nil {
						fmt.Printf("failed to read path %q: %v\n", p, err)
						return err
					}
					// ensure walked path is valid
					if !checkPathValid(p) {
						return nil
					}
					// add file as untracked
					if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
						var homeRelPath string
						homeRelPath, err = stripHome(p, home)
						if err != nil {
							return err
						}
						itemDiffs = append(itemDiffs, ItemDiff{
							homeRelPath: homeRelPath,
							path:        p,
							diff:        untracked,
						})
					}
					return nil
				})
			} else {
				homeRelPath, serr := stripHome(path, home)
				if serr != nil {
					fmt.Println("panic here")
					panic(serr)
				}
				itemDiffs = append(itemDiffs, ItemDiff{
					homeRelPath: homeRelPath,
					path:        path,
					diff:        untracked,
				})
			}
		}

		for _, p := range paths {
			_, f := filepath.Split(p)
			if f != "" && !stringInSlice(p, trackedPaths, true) {
				var homeRelPath string
				homeRelPath, err = stripHome(p, home)
				if err != nil {
					return
				}
				itemDiffs = append(itemDiffs, ItemDiff{
					homeRelPath: homeRelPath,
					path:        p,
					diff:        untracked,
				})
			}
		}
	}
	return itemDiffs, err
}

type ItemDiff struct {
	tagTitle    string
	noteTitle   string
	path        string
	homeRelPath string
	diff        string
	remote      gosn.Item
	local       string
}

func compare(tagTitle, path, home string, remote gosn.Item) ItemDiff {
	localStat, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Println("failed to close file:", path)
		}
	}()
	localBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	var homeRelPath string
	homeRelPath, err = stripHome(path, home)
	if err != nil {
		panic(err)
	}
	localStr := string(localBytes)
	if localStr != remote.Content.GetText() {
		var remoteUpdated time.Time
		remoteUpdated, err = time.Parse("2006-01-02T15:04:05.000Z", remote.UpdatedAt)
		if err != nil {
			log.Fatal(err)
		}
		// if content different and local file was updated more recently
		if localStat.ModTime().UTC().After(remoteUpdated.UTC()) || localStat.ModTime().UTC() == remoteUpdated.UTC() {
			return ItemDiff{
				tagTitle:    tagTitle,
				path:        path,
				homeRelPath: homeRelPath,
				noteTitle:   remote.Content.GetTitle(),
				diff:        localNewer,
				local:       string(localBytes),
				remote:      remote,
			}
		}
		// content different remote content was updated more recently
		return ItemDiff{
			tagTitle:    tagTitle,
			path:        path,
			homeRelPath: homeRelPath,
			noteTitle:   remote.Content.GetTitle(),
			diff:        remoteNewer,
			local:       string(localBytes),
			remote:      remote,
		}
	}
	// local and remote identical
	return ItemDiff{
		tagTitle:    tagTitle,
		path:        path,
		homeRelPath: homeRelPath,
		noteTitle:   remote.Content.GetTitle(),
		diff:        identical,
		local:       string(localBytes),
		remote:      remote,
	}
}
