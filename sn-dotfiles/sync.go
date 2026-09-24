package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/asdine/storm/v3"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/jonhadfield/gosn-v2/cache"
	"os"
	"strings"
	"time"

	"github.com/ryanuber/columnize"
)

var (
	HiWhite = color.New(color.FgHiWhite).SprintFunc()
)

// Sync compares local and remote items and then:
// - pulls remotes if locals are older or missing
// - pushes locals if locals are newer
func Sync(si SNDotfilesSyncInput, useStdErr bool) (so SyncOutput, err error) {
	if si.RootTag, err = ResolveRootTag(si.RootTag); err != nil {
		return
	}

	if err = checkPathsExist(si.Exclude); err != nil {
		return
	}

	if !si.Debug {
		prefix := HiWhite("syncing ")
		if _, err = os.Stat(si.Session.CacheDBPath); os.IsNotExist(err) {
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

	output, err := sync(syncInput{
		session: si.Session,
		home:    si.Home,
		paths:   si.Paths,
		exclude: si.Exclude,
		filter:  si.Filter,
		rootTag: si.RootTag,
		debug:   si.Debug,
		close:   false,
		dryRun:  si.DryRun,
	})

	return SyncOutput{
		NoPushed: output.noPushed,
		NoPulled: output.noPulled,
		Msg:      output.msg,
	}, err
}

func sync(input syncInput) (output syncOutput, err error) {
	// get populated db
	csi := cache.SyncInput{
		Session: input.session,
		Close:   false,
		// always fetch, as dotfiles may have changed on another machine since the last sync
		AlwaysSync: true,
	}

	var cso cache.SyncOutput
	cso, err = cache.Sync(csi)
	if err != nil {
		return
	}

	var remote tagsWithNotes
	remote, err = getTagsWithNotes(cso.DB, input.session, input.rootTag)
	if err != nil {
		return
	}

	err = checkNoteTagConflicts(remote, input.rootTag)
	if err != nil {
		return
	}

	output, err = syncDBwithFS(syncInput{
		db:      cso.DB,
		session: input.session,
		twn:     remote,
		home:    input.home,
		paths:   input.paths,
		exclude: input.exclude,
		filter:  input.filter,
		rootTag: input.rootTag,
		debug:   input.debug,
		dryRun:  input.dryRun})
	if err != nil {

		return
	}

	editors, err := getEditorAssociations(cso.DB, input.session, remote, input.home, input.rootTag)
	if err != nil {
		_ = cso.DB.Close()

		return output, err
	}

	if err = cso.DB.Close(); err != nil {
		return
	}

	output.msg += editorAssociationWarning(editors)

	// persist changes
	csi.Close = true
	_, err = cache.Sync(csi)

	return
}

type SNDotfilesSyncInput struct {
	Session        *cache.Session
	Home           string
	Paths, Exclude []string
	// Filter limits the sync to matching paths; nil syncs everything
	Filter *PathFilter
	// RootTag is the Standard Notes root tag for this sync; empty defaults to DotFilesTag
	RootTag  string
	PageSize int
	Debug    bool
	// DryRun reports what a sync would do without writing anything
	DryRun bool
}
type SyncOutput struct {
	NoPushed, NoPulled int
	Msg                string
}

func syncDBwithFS(si syncInput) (so syncOutput, err error) {
	if si.db == nil {
		panic("didn't get db sent to syncDBwithFS")
	}
	var itemDiffs []ItemDiff

	itemDiffs, err = compare(si.twn, si.home, si.paths, si.exclude, si.filter, si.rootTag, si.debug)
	if err != nil {
		if strings.Contains(err.Error(), "tags with notes not supplied") {
			err = errors.New("no remote dotfiles found")
		}

		return
	}

	var itemsToPush, itemsToPull, itemsUnchanged, itemsBinary []ItemDiff

	var itemsToSync bool
	for _, itemDiff := range itemDiffs {
		// check if itemDiff is for a path to be excluded
		if matchesPathsToExclude(si.home, itemDiff.homeRelPath, si.exclude) {
			debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | excluding: %s", itemDiff.homeRelPath))
			continue
		}

		switch itemDiff.diff {
		case localNewer:
			// A tracked file can become binary after it was added, and note
			// content is text, so pushing it would store corrupted content.
			var binary bool

			binary, err = isBinaryFile(itemDiff.path)
			if err != nil {
				return so, err
			}

			if binary {
				debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | skipping binary: %s", itemDiff.homeRelPath))
				itemsBinary = append(itemsBinary, itemDiff)

				continue
			}

			//addToDB
			debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | local %s is newer", itemDiff.homeRelPath))
			itemDiff.remote.Content.SetText(itemDiff.local)
			itemsToPush = append(itemsToPush, itemDiff)
			itemsToSync = true
		case localMissing:
			// createLocal
			debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | %s is missing", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		case remoteNewer:
			// createLocal
			debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | remote %s is newer", itemDiff.homeRelPath))
			itemsToPull = append(itemsToPull, itemDiff)
			itemsToSync = true
		case identical:
			// only reported by a dry run, which lists what it leaves alone
			itemsUnchanged = append(itemsUnchanged, itemDiff)
		}
	}

	// A dry run answers "what would sync do" from the same comparison a real
	// sync acts on, so it has to return before anything is written.
	if si.dryRun {
		so.noPushed = len(itemsToPush)
		so.noPulled = len(itemsToPull)
		so.msg = dryRunMsg(itemsToPush, itemsToPull, itemsUnchanged, itemsBinary)

		return so, nil
	}

	// check items to sync
	if !itemsToSync {
		if len(itemsBinary) > 0 {
			so.msg = fmt.Sprint(columnize.SimpleFormat(binaryLines(itemsBinary)))

			return
		}

		so.msg = fmt.Sprint(bold("nothing to do"))

		return
	}

	// addToDB
	if len(itemsToPush) > 0 {
		err = addToDB(si.db, si.session, itemsToPush, si.close)
		if err != nil {
			return
		}
		so.noPushed = len(itemsToPush)
	}

	res := make([]string, len(itemsToPush))
	strPushed := green("pushed")
	strPulled := green("pulled")

	for i, pushItem := range itemsToPush {
		line := fmt.Sprintf("%s | %s", bold(addDot(pushItem.homeRelPath)), strPushed)
		res[i] = line
	}

	// create local
	if err = createLocal(itemsToPull); err != nil {
		return
	}

	so.noPulled = len(itemsToPull)

	for _, pullItem := range itemsToPull {
		line := fmt.Sprintf("%s | %s\n", bold(addDot(pullItem.homeRelPath)), strPulled)
		res = append(res, line)
	}

	res = append(res, binaryLines(itemsBinary)...)

	so.msg = fmt.Sprint(columnize.SimpleFormat(res))

	return so, err
}

// binaryLines renders the paths a sync left alone because their content is not
// text, so they are not silently missing from its output.
func binaryLines(itemsBinary []ItemDiff) []string {
	lines := make([]string, 0, len(itemsBinary))

	for _, item := range itemsBinary {
		lines = append(lines, fmt.Sprintf("%s | %s\n", bold(addDot(item.homeRelPath)), yellow("skipped: binary file")))
	}

	return lines
}

// dryRunMsg renders what a real sync would have done to each path, followed by
// a summary making it plain that nothing was written.
func dryRunMsg(itemsToPush, itemsToPull, itemsUnchanged, itemsBinary []ItemDiff) string {
	if len(itemsToPush)+len(itemsToPull)+len(itemsUnchanged)+len(itemsBinary) == 0 {
		return fmt.Sprint(bold("nothing to do"))
	}

	lines := make([]string, 0, len(itemsToPush)+len(itemsToPull)+len(itemsUnchanged)+len(itemsBinary))

	for _, item := range itemsToPush {
		lines = append(lines, fmt.Sprintf("%s | %s", bold(addDot(item.homeRelPath)), green("would push")))
	}

	for _, item := range itemsToPull {
		lines = append(lines, fmt.Sprintf("%s | %s", bold(addDot(item.homeRelPath)), green("would pull")))
	}

	for _, item := range itemsUnchanged {
		lines = append(lines, fmt.Sprintf("%s | %s", bold(addDot(item.homeRelPath)), "unchanged"))
	}

	for _, item := range itemsBinary {
		lines = append(lines, fmt.Sprintf("%s | %s", bold(addDot(item.homeRelPath)), yellow("skipped: binary file")))
	}

	summary := fmt.Sprintf("dry run: nothing was written (%d to push, %d to pull)",
		len(itemsToPush), len(itemsToPull))

	return fmt.Sprintf("%s\n\n%s", columnize.SimpleFormat(lines), bold(summary))
}

type syncInput struct {
	db             *storm.DB
	session        *cache.Session
	twn            tagsWithNotes
	home           string
	paths, exclude []string
	filter         *PathFilter
	rootTag        string
	debug          bool
	close          bool
	dryRun         bool
}

type syncOutput struct {
	noPushed, noPulled int
	msg                string
}

func ensureTrailingPathSep(in string) string {
	if strings.HasSuffix(in, string(os.PathSeparator)) {
		return in
	}

	return in + string(os.PathSeparator)
}

func matchesPathsToExclude(home, path string, pathsToExclude []string) bool {
	for _, pte := range pathsToExclude {
		homeStrippedPath := stripHome(pte, home)
		// return match if Paths match exactly
		if homeStrippedPath == path {
			return true
		}
		// return match if pte is a parent of the path
		if strings.HasPrefix(path, ensureTrailingPathSep(homeStrippedPath)) {
			return true
		}
	}

	return false
}
