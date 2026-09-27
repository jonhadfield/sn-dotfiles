package sndotfiles

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/asdine/storm/v3"
	"github.com/fatih/color"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/ryanuber/columnize"
)

var (
	HiWhite = color.New(color.FgHiWhite).SprintFunc()
)

// Sync compares local and remote items and then:
// - pulls remotes if locals are older or missing
// - pushes locals if locals are newer
// With Interactive set, differing files are prompted for local/remote/skip
// instead of applying last-write-wins automatically.
func Sync(si SNDotfilesSyncInput, useStdErr bool) (so SyncOutput, err error) {
	if si.DryRun && si.Interactive {
		return so, errors.New("--interactive cannot be used with --dry-run")
	}

	if si.RootTag, err = ResolveRootTag(si.RootTag); err != nil {
		return
	}

	if err = checkPathsExist(si.Exclude); err != nil {
		return
	}

	// Skip the spinner in interactive mode so prompts are not overwritten.
	if !si.Debug && !si.Interactive {
		stop := startBusySpinner(si.Session.CacheDBPath)
		defer stop()
	}

	chooser := si.ChooseConflict
	if si.Interactive && chooser == nil {
		chooser = defaultConflictChooser
	}

	output, err := sync(syncInput{
		session:        si.Session,
		home:           si.Home,
		paths:          si.Paths,
		exclude:        si.Exclude,
		filter:         si.Filter,
		rootTag:        si.RootTag,
		debug:          si.Debug,
		close:          false,
		dryRun:         si.DryRun,
		interactive:    si.Interactive,
		chooseConflict: chooser,
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
		db:             cso.DB,
		session:        input.session,
		twn:            remote,
		home:           input.home,
		paths:          input.paths,
		exclude:        input.exclude,
		filter:         input.filter,
		rootTag:        input.rootTag,
		debug:          input.debug,
		dryRun:         input.dryRun,
		interactive:    input.interactive,
		chooseConflict: input.chooseConflict,
	})
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
	// Interactive prompts for each differing file instead of last-write-wins
	Interactive bool
	// ChooseConflict overrides the default stdin prompt when Interactive is set.
	// Tests inject a chooser; CLI usage leaves it nil.
	ChooseConflict ConflictChooser
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

	var itemsToPush, itemsToPull, itemsUnchanged, itemsBinary, itemsSkipped []ItemDiff

	var itemsToSync bool
	for _, itemDiff := range itemDiffs {
		// check if itemDiff is for a path to be excluded
		if matchesPathsToExclude(si.home, itemDiff.homeRelPath, si.exclude) {
			debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | excluding: %s", itemDiff.homeRelPath))
			continue
		}

		switch itemDiff.diff {
		case localNewer, remoteNewer:
			// A tracked file can become binary after it was added. Note content
			// is text, so never prompt to push something that cannot be stored.
			if itemDiff.diff == localNewer {
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
			}

			var action SyncChoice
			action, err = resolveSyncAction(si, itemDiff)
			if err != nil {
				return so, err
			}

			switch action {
			case SyncChoiceLocal:
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

				debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | pushing %s", itemDiff.homeRelPath))
				itemDiff.remote.Content.SetText(itemDiff.local)
				itemsToPush = append(itemsToPush, itemDiff)
				itemsToSync = true
			case SyncChoiceRemote:
				debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | pulling %s", itemDiff.homeRelPath))
				itemsToPull = append(itemsToPull, itemDiff)
				itemsToSync = true
			case SyncChoiceSkip:
				debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | skipping %s", itemDiff.homeRelPath))
				itemsSkipped = append(itemsSkipped, itemDiff)
			}
		case localMissing:
			// createLocal
			debugPrint(si.debug, fmt.Sprintf("syncDBwithFS | %s is missing", itemDiff.homeRelPath))
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
		lines := append(binaryLines(itemsBinary), skippedLines(itemsSkipped)...)
		if len(lines) > 0 {
			so.msg = fmt.Sprint(columnize.SimpleFormat(lines))

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
	res = append(res, skippedLines(itemsSkipped)...)

	so.msg = fmt.Sprint(columnize.SimpleFormat(res))

	return so, err
}

// resolveSyncAction returns last-write-wins, or asks the chooser when interactive.
func resolveSyncAction(si syncInput, itemDiff ItemDiff) (SyncChoice, error) {
	defaultAction := SyncChoiceLocal
	if itemDiff.diff == remoteNewer {
		defaultAction = SyncChoiceRemote
	}

	if !si.interactive {
		return defaultAction, nil
	}

	if si.chooseConflict == nil {
		return SyncChoiceSkip, errors.New("interactive sync requires a conflict chooser")
	}

	return si.chooseConflict(itemDiff, defaultAction)
}

func skippedLines(items []ItemDiff) []string {
	lines := make([]string, 0, len(items))

	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s | %s\n", bold(addDot(item.homeRelPath)), yellow("skipped")))
	}

	return lines
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
	interactive    bool
	chooseConflict ConflictChooser
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
