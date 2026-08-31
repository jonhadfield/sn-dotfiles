package sndotfiles

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/jonhadfield/findexec"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/items"
)

const (
	localMissing = "local missing"
	localNewer   = "local newer"
	remoteNewer  = "remote newer"
	untracked    = "untracked"
	identical    = "identical"
)

func Diff(sess *cache.Session, home string, paths []string, pageSize int, close, useStdErr bool) (diffs []ItemDiff, msg string, err error) {
	debugPrint(sess.Debug, fmt.Sprintf("Diff | %d paths", len(paths)))

	if !sess.Debug {
		prefix := HiWhite("syncing ")
		if _, sErr := os.Stat(sess.CacheDBPath); os.IsNotExist(sErr) {
			prefix = HiWhite("initializing ")
		}

		s := spinner.New(spinner.CharSets[SpinnerCharSet], SpinnerDelay*time.Millisecond, spinner.WithWriter(os.Stdout))
		if useStdErr {
			s = spinner.New(spinner.CharSets[SpinnerCharSet], SpinnerDelay*time.Millisecond, spinner.WithWriter(os.Stderr))
		}

		s.Prefix = prefix
		s.Start()
		defer s.Stop()
	}

	// get populated db
	si := cache.SyncInput{
		Session: sess,
		Close:   false,
	}

	var cso cache.SyncOutput

	cso, err = cache.Sync(si)
	if err != nil {
		return
	}

	var remote tagsWithNotes

	remote, err = getTagsWithNotes(cso.DB, sess)

	if cErr := cso.DB.Close(); cErr != nil && err == nil {
		err = cErr
	}

	sess.CacheDB = nil

	if err != nil {
		return diffs, msg, err
	}

	return diff(remote, home, paths, sess.Debug)
}

type ItemDiff struct {
	tagTitle    string
	noteTitle   string
	path        string
	homeRelPath string
	diff        string
	remote      items.Note
	local       string
}

func diff(twn tagsWithNotes, home string, paths []string, debug bool) (diffs []ItemDiff, msg string, err error) {
	debugPrint(debug, fmt.Sprintf("diff | %d remote items", len(twn)))

	err = checkNoteTagConflicts(twn)
	if err != nil {
		return
	}

	if len(twn) == 0 {
		msg = "no dotfiles being tracked"
		return
	}

	if len(paths) == 0 {
		debugPrint(debug, fmt.Sprint("diff | calling compare without any Paths"))
	} else {
		debugPrint(debug, fmt.Sprintf("diff | calling compare with Paths: %s", strings.Join(paths, ",")))
	}

	diffs, err = compare(twn, home, paths, []string{}, debug)
	if err != nil {
		return diffs, msg, err
	}

	debugPrint(debug, fmt.Sprintf("compare | %d diffs generated", len(diffs)))

	if len(diffs) == 0 {
		return diffs, msg, err
	}

	diffBinary := findexec.Find("diff", "")
	if diffBinary == "" {
		err = errors.New("failed to find compare binary")
		return
	}

	var differencesFound bool
	// getTagsWithNotes tempdir
	tempDir := os.TempDir()
	if !strings.HasSuffix(tempDir, string(os.PathSeparator)) {
		tempDir += string(os.PathSeparator)
	}

	differencesFound, err = processContentDiffs(diffs, tempDir, diffBinary)
	if err != nil {
		return
	}

	if !differencesFound {
		msg = "no differences found"
	}

	return diffs, msg, err
}

func processContentDiffs(diffs []ItemDiff, tempDir, diffBinary string) (differencesFound bool, err error) {
	for _, diff := range diffs {
		localContent := diff.local

		remoteContent := diff.remote.Content.GetText()
		if localContent == remoteContent {
			continue
		}

		differencesFound = true

		var out []byte

		out, err = diffContent(diffBinary, tempDir, localContent, remoteContent)
		if err != nil {
			return
		}

		fmt.Println(bold(diff.homeRelPath))
		fmt.Println(string(out))
	}

	return differencesFound, err
}

// diffContent writes local and remote content to temporary files and returns
// the output of running diffBinary over them.
func diffContent(diffBinary, tempDir, local, remote string) (out []byte, err error) {
	uuid := items.GenUUID()

	f1path := fmt.Sprintf("%ssn-dotfiles-compare-%s-f1", tempDir, uuid)
	f2path := fmt.Sprintf("%ssn-dotfiles-compare-%s-f2", tempDir, uuid)

	defer func() {
		for _, p := range []string{f1path, f2path} {
			if rErr := os.Remove(p); rErr != nil && !os.IsNotExist(rErr) && err == nil {
				err = rErr
			}
		}
	}()

	if err = writeLocal(f1path, local); err != nil {
		return
	}

	if err = writeLocal(f2path, remote); err != nil {
		return
	}

	out, oErr := exec.Command(diffBinary, f1path, f2path).CombinedOutput()

	// diff exits 0 when the files match and 1 when they differ; anything else
	// means diff itself failed.
	var exitError *exec.ExitError
	if oErr != nil && (!errors.As(oErr, &exitError) || exitError.ExitCode() > 1) {
		return out, fmt.Errorf("failed to compare %q with %q: %w", f1path, f2path, oErr)
	}

	return out, err
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
			return fmt.Errorf("failed to read path: %s", p)
		}
	}

	return nil
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
		debugPrint(debug, fmt.Sprintf("compare | diffing path: %s", stripHome(path, home)))

		if StringInSlice(path, existingRemoteEquivalentPaths, true) {
			continue
		}

		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			debugPrint(debug, fmt.Sprintf("compare | walking path: %s", path))

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
				if v, err := pathValid(p); !v {
					return err
				}
				// add file as untracked
				if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
					debugPrint(debug, fmt.Sprintf("compare | file is untracked: %s", p))
					homeRelPath := stripHome(p, home)
					itemDiffs = append(itemDiffs, ItemDiff{
						homeRelPath: homeRelPath,
						path:        p,
						diff:        untracked,
					})
				}
				return nil
			})
			if err != nil {
				return
			}
		} else {
			homeRelPath := stripHome(path, home)
			debugPrint(debug, fmt.Sprintf("compare | file is untracked: %s", path))

			itemDiffs = append(itemDiffs, ItemDiff{
				homeRelPath: homeRelPath,
				path:        path,
				diff:        untracked,
			})
		}
	}

	return itemDiffs
}
