package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonhadfield/findexec"
	"github.com/jonhadfield/gosn"
)

const (
	localMissing = "local missing"
	localNewer   = "local newer"
	remoteNewer  = "remote newer"
	untracked    = "untracked"
	identical    = "identical"
)

func Diff(session gosn.Session, home string, paths []string, debug bool) (diffs []ItemDiff, msg string, err error) {
	remote, err := get(session)
	if err != nil {
		return diffs, msg, err
	}
	return appDiff(remote, home, paths, debug)
}

func appDiff(twn tagsWithNotes, home string, paths []string, debug bool) (diffs []ItemDiff, msg string, err error) {
	debugPrint(debug, fmt.Sprintf("diff | %d remote items", len(twn)))
	err = preflight(twn, paths)
	if err != nil {
		return
	}
	if len(twn) == 0 {
		msg = "no dotfiles being tracked"
		return
	}
	if len(paths) == 0 {
		debugPrint(debug, fmt.Sprint("appDiff | calling diff without any paths"))
	} else {
		debugPrint(debug, fmt.Sprintf("appDiff | calling diff with paths: %s", strings.Join(paths, ",")))
	}

	diffs, err = diff(twn, home, paths, debug)
	if err != nil {
		return diffs, msg, err
	}
	debugPrint(debug, fmt.Sprintf("diff | %d diffs generated", len(diffs)))
	//var lines []string
	if len(diffs) == 0 {
		return diffs, msg, err
	}
	diffBinary := findexec.Find("diff", "")
	if diffBinary == "" {
		err = errors.New("failed to find diff binary")
		return
	}
	// get tempdir
	tempDir := os.TempDir()
	for _, diff := range diffs {
		localContent := diff.local
		remoteContent := diff.remote.Content.GetText()
		if localContent != remoteContent {
			// write local and remote content to temporary files
			var f1, f2 *os.File
			uuid := gosn.GenUUID()
			f1path := fmt.Sprintf("%ssn-dotfiles-diff-%s-f1", tempDir, uuid)
			f2path := fmt.Sprintf("%ssn-dotfiles-diff-%s-f2", tempDir, uuid)
			f1, err = os.Create(f1path)
			if err != nil {
				return
			}
			f2, err = os.Create(f2path)
			if err != nil {
				return
			}
			_, err = f1.WriteString(diff.local)
			_, err = f2.WriteString(diff.remote.Content.GetText())
			cmd := exec.Command(diffBinary, f1path, f2path)
			var out []byte
			out, err = cmd.CombinedOutput()
			if f1DelErr := os.Remove(f1path); f1DelErr != nil {
				err = f1DelErr
				return
			}
			if f2DelErr := os.Remove(f2path); f2DelErr != nil {
				err = f2DelErr
				return
			}
			var exitCode int
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				}
			}
			if exitCode == 2 {
				panic(fmt.Sprintf("failed to diff: '%s' with '%s'", f1path, f2path))
			}

			bold := color.New(color.Bold).SprintFunc()
			fmt.Println(bold(diff.remote.Content.GetTitle()))
			fmt.Println(string(out))
		}
	}
	return diffs, msg, err
}

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
		rel, err := filepath.Rel(paths[i], note)
		if err == nil && !strings.HasPrefix(rel, "../") {
			return true
		}
	}
	return false
}

func checkPathsExist(paths []string) error {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil || os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func diffRemoteWithLocalFS(remote tagsWithNotes, paths []string, home string, debug bool) (itemDiffs []ItemDiff, remotePaths []string, err error) {
	// loop through remotes to generate a list of diffs for:
	// - existing local and remotes
	// - missing local files
	// also get a list of remotes that should have locals
	for _, twn := range remote {
		// only do a diff if path equals translated tag
		tagTitle := twn.tag.Content.GetTitle()
		var dir string
		dir, _, err = tagTitleToFSDIR(twn.tag.Content.GetTitle(), home)
		if err != nil {
			return
		}
		debugPrint(debug, fmt.Sprintf("diff | tag title: %s is path: <Home>/%s", tagTitle, stripHome(dir, home)))
		// if Paths were supplied, then check the determined dir is a prefix of one of those
		if len(paths) > 0 && !pathIsPrefixOfPaths(dir, paths) {
			continue
		}

		// loop through notes for the tag and compare content of any with matching file
		// log each matching path so we can later walk them to discover untracked files
		for _, d := range twn.notes {
			fullPath := fmt.Sprintf("%s%s", dir, d.Content.GetTitle())
			// skip note if exact path is not specified and does not have prefix of total path
			if len(paths) > 0 && !noteInPaths(dir+d.Content.GetTitle(), paths) {
				continue
			}
			if !localExists(fullPath) {
				// local path matching tag+note doesn't exist so set as 'local missing'
				debugPrint(debug, fmt.Sprintf("diff | local not found: <Home>/%s", stripHome(fullPath, home)))
				homeRelPath := stripHome(fullPath, home)
				itemDiffs = append(itemDiffs, ItemDiff{
					tagTitle:    tagTitle,
					homeRelPath: homeRelPath,
					path:        fullPath,
					diff:        localMissing,
					noteTitle:   d.Content.GetTitle(),
					remote:      d,
				})
			} else {
				// local does exist, so compare and store generated diff
				debugPrint(debug, fmt.Sprintf("diff | local found: <Home>/%s", stripHome(fullPath, home)))
				remotePaths = append(remotePaths, fullPath)
				itemDiffs = append(itemDiffs, compare(tagTitle, fullPath, home, d, debug))
			}
		}
	}
	return itemDiffs, remotePaths, err
}

func diff(remote tagsWithNotes, home string, paths []string, debug bool) (diffs []ItemDiff, err error) {
	debugPrint(debug, fmt.Sprintf("diff | Home: %s", home))
	debugPrint(debug, fmt.Sprintf("diff | %d Paths supplied", len(paths)))

	// fail immediately if remote or Paths are empty
	if len(remote) == 0 {
		return nil, fmt.Errorf("tags with notes not supplied")
	}
	// if Paths specified, check all of them exist before continuing
	if len(paths) > 0 {
		if err := checkPathsExist(paths); err != nil {
			return nil, err
		}
	}

	var itemDiffs []ItemDiff
	var remotePaths []string
	// check remotes against local filesystem
	itemDiffs, remotePaths, err = diffRemoteWithLocalFS(remote, paths, home, debug)
	if err != nil {
		return
	}

	// if Paths specified, then discover those that are untracked
	// by comparing with existing remote equivalent Paths
	if len(paths) > 0 {
		itemDiffs = append(itemDiffs, findUntracked(paths, remotePaths, home, debug)...)
	}
	return itemDiffs, err
}

func tagExists(title string, twn tagsWithNotes) bool {
	for _, twn := range twn {
		if twn.tag.Content.GetTitle() == title {
			return true
		}
	}
	return false
}

func findUntracked(paths, existingRemoteEquivalentPaths []string, home string, debug bool) (itemDiffs []ItemDiff) {
	// if path is directory, then walk to generate list of additional Paths
	for _, path := range paths {
		debugPrint(debug, fmt.Sprintf("diff | diffing path: %s", stripHome(path, home)))
		if StringInSlice(path, existingRemoteEquivalentPaths, true) {
			continue
		}
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			debugPrint(debug, fmt.Sprintf("diff | walking path: %s", path))
			err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				// don't check tracked Paths
				if StringInSlice(p, existingRemoteEquivalentPaths, true) {
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
					debugPrint(debug, fmt.Sprintf("diff | file is untracked: %s", p))
					homeRelPath := stripHome(p, home)
					itemDiffs = append(itemDiffs, ItemDiff{
						homeRelPath: homeRelPath,
						path:        p,
						diff:        untracked,
					})
				}
				return nil
			})
		} else {
			homeRelPath := stripHome(path, home)
			debugPrint(debug, fmt.Sprintf("diff | file is untracked: %s", path))

			itemDiffs = append(itemDiffs, ItemDiff{
				homeRelPath: homeRelPath,
				path:        path,
				diff:        untracked,
			})
		}
	}

	return itemDiffs
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

func compare(tagTitle, path, home string, remote gosn.Item, debug bool) ItemDiff {
	debugPrint(debug, fmt.Sprintf("compare | title: %s path: <Home>/%s", tagTitle, stripHome(path, home)))
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
	homeRelPath := stripHome(path, home)
	localStr := string(localBytes)
	if localStr != remote.Content.GetText() {
		var remoteUpdated time.Time
		remoteUpdated, err = time.Parse("2006-01-02T15:04:05.000Z", remote.UpdatedAt)
		if err != nil {
			log.Fatal(err)
		}
		debugPrint(debug, fmt.Sprintf("compare | remote updated UTC): %v", remoteUpdated.UTC()))
		// if content different and local file was updated more recently
		debugPrint(debug, fmt.Sprintf("compare | local updated UTC): %v", localStat.ModTime().UTC().Format("2006-01-02T15:04:05.000Z")))
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
