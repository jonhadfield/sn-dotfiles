package sndotfiles

import (
	"bytes"
	"fmt"
	"github.com/asdine/storm/v3"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	gosn "github.com/jonhadfield/gosn-v2/items"
	"github.com/jonhadfield/gosn-v2/session"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

func debugPrint(show bool, msg string) {
	if show {
		if len(msg) > maxDebugChars {
			msg = msg[:maxDebugChars] + "..."
		}

		log.Println(SNAppName, "|", msg)
	}
}

func addDot(in string) string {
	if !strings.HasPrefix(in, ".") {
		return fmt.Sprintf(".%s", in)
	}

	return in
}

func stripDot(in string) string {
	if strings.HasPrefix(in, ".") {
		return in[1:]
	}

	return in
}

func localExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func stripHome(in, home string) string {
	if home != "" && strings.HasPrefix(in, home) {
		return in[len(home)+1:]
	}

	return in
}

func addToDB(db *storm.DB, session *cache.Session, itemDiffs []ItemDiff, close bool) (err error) {
	var dItems gosn.Items
	for i := range itemDiffs {
		dItems = append(dItems, &itemDiffs[i].remote)
	}

	if dItems == nil {
		err = errors.New("no items to addToDB")
		return
	}

	return cache.SaveItems(session, db, dItems, close)
}

func getTagIfExists(name string, twn tagsWithNotes) (tag gosn.Tag, found bool) {
	for _, x := range twn {
		if name == x.tag.Content.GetTitle() {
			return x.tag, true
		}
	}

	return tag, false
}

func createMissingTags(db *storm.DB, session *cache.Session, pt string, twn tagsWithNotes) (newTags gosn.Tags, err error) {
	var fts []string

	ts := strings.Split(pt, ".")
	for x, t := range ts {
		switch {
		case x == 0:
			fts = append(fts, t)
		case x+1 == len(ts):
			a := strings.Join(fts[len(fts)-1:], ".") + "." + t
			fts = append(fts, a)
		default:
			a := strings.Join(fts[len(fts)-1:], ".") + "." + t
			fts = append(fts, a)
		}
	}

	itemsToPush := gosn.Items{}

	for _, f := range fts {
		_, found := getTagIfExists(f, twn)
		if !found {
			var nt gosn.Tag

			nt, err = createTag(f)
			if err != nil {
				return
			}

			itemsToPush = append(itemsToPush, &nt)
		}
	}

	err = cache.SaveItems(session, db, itemsToPush, false)
	if err != nil {
		return
	}

	return itemsToPush.Tags(), err
}

func pushAndTag(db *storm.DB, session *cache.Session, tim map[string]gosn.Items, twn tagsWithNotes) (tagsPushed, notesPushed int, err error) {
	// create missing tags first to create a new tim
	itemsToPush := gosn.Items{}
	for potentialTag, notes := range tim {
		existingTag, found := getTagIfExists(potentialTag, twn)
		if found {
			// if tag exists then just add references to the note
			var newReferences gosn.ItemReferences

			for _, note := range notes {
				itemsToPush = append(itemsToPush, note)
				newReferences = append(newReferences, gosn.ItemReference{
					UUID:        note.GetUUID(),
					ContentType: "Note",
				})
			}

			existingTag.Content.UpsertReferences(newReferences)
			itemsToPush = append(itemsToPush, &existingTag)
		} else {
			// need to create tag
			var newTags gosn.Tags
			newTags, err = createMissingTags(db, session, potentialTag, twn)
			if err != nil {
				return
			}
			// create a new item reference for each note to be tagged
			var newReferences gosn.ItemReferences
			for _, note := range notes {
				itemsToPush = append(itemsToPush, note)
				newReferences = append(newReferences, gosn.ItemReference{
					UUID:        note.GetUUID(),
					ContentType: "Note",
				})
			}
			newTag := newTags[len(newTags)-1]
			newTag.Content.UpsertReferences(newReferences)
			itemsToPush = append(itemsToPush, &newTag)

			// add to twn so we don't getTagsWithNotes duplicates
			twn = append(twn, tagWithNotes{
				tag:   newTag,
				notes: notes.Notes(),
			})
			for x := 0; x < len(newTags)-1; x++ {
				twn = append(twn, tagWithNotes{
					tag:   newTags[x],
					notes: nil,
				})
			}
		}
	}
	if len(itemsToPush) == 0 {
		return 0, 0, nil
	}

	err = cache.SaveItems(session, db, itemsToPush, true)
	tagsPushed, notesPushed = getItemCounts(itemsToPush)

	return tagsPushed, notesPushed, err
}

func getItemCounts(items gosn.Items) (tags, notes int) {
	return len(items.Tags()), len(items.Notes())
}

func createTag(name string) (gosn.Tag, error) {
	return gosn.NewTag(name, nil)
}

func createLocal(itemDiffs []ItemDiff) error {
	for _, item := range itemDiffs {
		dir, _ := filepath.Split(item.path)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}

		f, err := os.Create(item.path)
		if err != nil {
			return err
		}

		_, err = f.WriteString(item.remote.Content.GetText())
		if err != nil {
			f.Close()
			return err
		}
	}

	return nil
}

func getPathType(path string) (res string, err error) {
	var stat os.FileInfo

	stat, err = os.Stat(path)
	if err != nil {
		return
	}

	switch mode := stat.Mode(); {
	case mode.IsDir():
		res = "dir"
	case mode.IsRegular():
		res = "file"
	}

	return
}

func noteInNotes(item gosn.Note, items gosn.Notes) bool {
	for _, i := range items {
		if i.GetUUID() == item.GetUUID() {
			return true
		}
	}

	return false
}

// getAllTagsWithoutNotes finds all tags that no longer have notes
// (doesn't check tags that are empty after child tag(s) removed)
func getAllTagsWithoutNotes(twn tagsWithNotes, deletedNotes gosn.Notes, debug bool) (tagsWithoutNotes []string) {
	// getTagsWithNotes a map of all tags and notes, minus the notes to delete
	res := make(map[string]int)
	// initialise map with 0 count
	for _, x := range twn {
		res[x.tag.Content.GetTitle()] = 0
	}
	// getTagsWithNotes a count of notes for each tag
	for _, t := range twn {
		debugPrint(debug, fmt.Sprintf("getAllTagsWithoutNotes | tag: %s", t.tag.Content.GetTitle()))

		// generate list of tags to reduce later
		for _, n := range t.notes {
			if !noteInNotes(n, deletedNotes) {
				res[t.tag.Content.GetTitle()]++
			}
		}
	}
	// create list of tags without notes
	for tn, count := range res {
		if count == 0 {
			debugPrint(debug, fmt.Sprintf("getAllTagsWithoutNotes | tag: %s has no notes", tn))
			tagsWithoutNotes = append(tagsWithoutNotes, tn)
		}
	}

	return
}

func removeStringFromSlice(item string, slice []string) (updatedSlice []string) {
	for i := range slice {
		if item != slice[i] {
			updatedSlice = append(updatedSlice, slice[i])
		}
	}

	return
}

// findEmptyTags takes a set of tags with notes and a list of notes being deleted
// in order to find all tags that are already empty or will be empty once the notes are deleted
func findEmptyTags(twn tagsWithNotes, deletedNotes gosn.Notes, rootTag string, debug bool) gosn.Tags {
	rootTag = normalizeRootTag(rootTag)

	// getTagsWithNotes a list of tags without notes (including those that have just become noteless)
	allTagsWithoutNotes := getAllTagsWithoutNotes(twn, deletedNotes, debug)
	debugPrint(debug, fmt.Sprintf("findEmptyTags | allTagsWithoutNotes: %s", allTagsWithoutNotes))

	// generate a map of tag child counts
	allTagsChildMap := make(map[string][]string)

	var tagsToRemove []string

	var allRootChildTags []string
	// loop through all identified tags with their associated notes and generate a map of them
	// for each tag, the last item is the child
	for _, atwn := range twn {
		if strings.HasPrefix(atwn.tag.Content.GetTitle(), rootTag+".") {
			allRootChildTags = append(allRootChildTags, atwn.tag.Content.GetTitle())
		}

		tagTitle := atwn.tag.Content.GetTitle()
		splitTag := strings.Split(tagTitle, ".")

		if strings.Contains(tagTitle, ".") {
			firstPart := splitTag[:len(splitTag)-1]
			lastPart := splitTag[len(splitTag)-1:]
			allTagsChildMap[strings.Join(firstPart, ".")] = append(allTagsChildMap[strings.Join(firstPart, ".")], strings.Join(lastPart, "."))
		}
	}

	debugPrint(debug, fmt.Sprintf("findEmptyTags | allTagsChildMap: %s", allTagsChildMap))
	debugPrint(debug, fmt.Sprintf("findEmptyTags | allTagsWithoutNotes: %s", allTagsWithoutNotes))

	// removeFromDB tags without notes and without children
	for {
		var changeMade bool

		// loop through all tags and children looking for those without child tags
		for k, v := range allTagsChildMap {
			debugPrint(debug, fmt.Sprintf("findEmptyTags | tag: %s children: %s", k, v))

			for _, i := range v {
				completeTag := k + "." + i
				// check if noteless tag exists
				debugPrint(debug, fmt.Sprintf("findEmptyTags | completeTag: %s", completeTag))

				if StringInSlice(completeTag, allTagsWithoutNotes, true) {
					debugPrint(debug, fmt.Sprintf("findEmptyTags | completeTag: %s exists in: %s", completeTag, allTagsWithoutNotes))

					// check if tag still has children
					if len(allTagsChildMap[completeTag]) == 0 {
						debugPrint(debug, fmt.Sprintf("findEmptyTags | removing: %s from %s", i, v))
						allTagsChildMap[k] = removeStringFromSlice(i, v)
						debugPrint(debug, fmt.Sprintf("findEmptyTags | allTagsChildMap is now: %s", allTagsChildMap))

						tagsToRemove = append(tagsToRemove, k+"."+i)
						changeMade = true
					}
				}
			}
		}

		if !changeMade {
			break
		}
	}

	tagsToRemove = dedupe(tagsToRemove)

	// now removeFromDB root tag if it has no children
	debugPrint(debug, fmt.Sprintf("findEmptyTags | tagsToRemove: %s", tagsToRemove))
	debugPrint(debug, fmt.Sprintf("findEmptyTags | allRootChildTags: %s", allRootChildTags))

	// Remove the root tag only when it is left holding nothing: no notes of its
	// own, and every child tag going too. Comparing the lengths alone matched
	// when both were empty, so a remove that found nothing to do still took the
	// root tag with it, and everything under it stopped being tracked.
	rootTagIsEmpty := StringInSlice(rootTag, allTagsWithoutNotes, true)

	if rootTagIsEmpty && len(tagsToRemove) == len(allRootChildTags) {
		tagsToRemove = append(tagsToRemove, rootTag)
		debugPrint(debug, fmt.Sprintf("findEmptyTags | removing '%s' tag as it has no notes and all children are being removed", rootTag))
	}

	debugPrint(debug, fmt.Sprintf("findEmptyTags | tags to removeFromDB (deduped): %s", tagsToRemove))

	debugPrint(debug, fmt.Sprintf("findEmptyTags | total to removeFromDB: %d", len(tagsToRemove)))

	return tagTitlesToTags(tagsToRemove, twn)
}

func tagTitlesToTags(tagTitles []string, twn tagsWithNotes) (res gosn.Tags) {
	for _, t := range twn {
		if StringInSlice(t.tag.Content.GetTitle(), tagTitles, true) {
			res = append(res, t.tag)
		}
	}

	return
}

func getNotesToRemove(path, home string, twn tagsWithNotes, rootTag string, debug bool) (homeRelPath string, pathsToRemove []string, res gosn.Notes) {
	rootTag = normalizeRootTag(rootTag)

	pathType, err := getPathType(path)
	if err != nil {
		return
	}

	homeRelPath = stripHome(path, home)
	remoteEquiv := homeRelPath

	debugPrint(debug, fmt.Sprintf("getNotesToRemove | path: '%s': %s", path, remoteEquiv))

	// getTagsWithNotes item tags from remoteEquiv by stripping <rootTag> and filename from remoteEquiv
	var noteTag, noteTitle string

	debugPrint(debug, fmt.Sprintf("getNotesToRemove | path type: %s", pathType))

	if pathType != "dir" {
		// split between tag and title if remote equivalent doesn't contain slash
		if strings.Contains(remoteEquiv, string(os.PathSeparator)) {
			remoteEquiv = stripDot(remoteEquiv)
			noteTag, noteTitle = filepath.Split(remoteEquiv)
			noteTag = rootTag + "." + strings.ReplaceAll(noteTag[:len(noteTag)-1], string(os.PathSeparator), ".")
		} else {
			noteTag = rootTag
			noteTitle = remoteEquiv
		}

		// check if remote exists
		for _, t := range twn {
			if t.tag.Content.GetTitle() == noteTag {
				for _, note := range t.notes {
					if note.Content.GetTitle() == noteTitle {
						res = append(res, note)
						pathsToRemove = append(pathsToRemove, homeRelPath)
					}
				}
			}
		}
	} else {
		// tag specified so find all notes matching tag and tags underneath
		remoteEquiv = stripDot(remoteEquiv)
		debugPrint(debug, fmt.Sprintf("getNotesToRemove | remoteEquiv: %s", remoteEquiv))

		// strip trailing slash if provided
		if strings.HasSuffix(remoteEquiv, string(os.PathSeparator)) {
			remoteEquiv = remoteEquiv[:len(remoteEquiv)-1]
		}

		// replace path separatators with dots
		remoteEquiv = strings.ReplaceAll(remoteEquiv, string(os.PathSeparator), ".")

		noteTag = rootTag + "." + remoteEquiv
		debugPrint(debug, fmt.Sprintf("getNotesToRemove | find notes matching tag: %s", noteTag))

		// find notes matching tag
		for _, t := range twn {
			tagTitle := t.tag.Content.GetTitle()
			var tp string
			tp, err = tagTitleToFSDir(tagTitle, home, rootTag)
			if err != nil {
				return
			}
			tp = stripHome(tp, home)

			if t.tag.Content.GetTitle() == noteTag || strings.HasPrefix(t.tag.Content.GetTitle(), noteTag+".") {
				for _, note := range t.notes {
					pathsToRemove = append(pathsToRemove, fmt.Sprintf("%s%s", tp, note.Content.GetTitle()))
					{
						res = append(res, note)
					}
				}
			}
		}
	}
	// dedupe in case items discovered multiple times
	if res != nil {
		res.DeDupe()
	}

	return homeRelPath, pathsToRemove, res
}

func noteWithTagExists(tag, name string, twn tagsWithNotes) (count int) {
	for _, t := range twn {
		if t.tag.Content.GetTitle() == tag {
			for _, note := range t.notes {
				if note.Content.GetTitle() == name {
					count++
				}
			}
		}
	}

	return count
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}

	sort.Strings(in)

	j := 0

	for i := 1; i < len(in); i++ {
		if in[j] == in[i] {
			continue
		}
		j++

		in[j] = in[i]
	}

	return in[:j+1]
}

func tagTitleToFSDir(title, home, rootTag string) (path string, err error) {
	if title == "" {
		err = errors.New("tag title required")
		return
	}

	if home == "" {
		err = errors.New("home directory required")
		return
	}

	rootTag = normalizeRootTag(rootTag)

	if title != rootTag && !strings.HasPrefix(title, rootTag+".") {
		return
	}

	if title == rootTag {
		return home + string(os.PathSeparator), nil
	}

	a := title[len(rootTag)+1:]
	b := strings.ReplaceAll(a, ".", string(os.PathSeparator))
	c := addDot(b)

	return home + string(os.PathSeparator) + c + string(os.PathSeparator), err
}

func pathToTag(homeRelPath, rootTag string) string {
	rootTag = normalizeRootTag(rootTag)
	// prepend root tag
	r := rootTag + homeRelPath
	// replace path separators with dots
	r = strings.ReplaceAll(r, string(os.PathSeparator), ".")
	if strings.HasSuffix(r, ".") {
		return r[:len(r)-1]
	}

	return r
}

func isUnencryptedSession(in string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if len(strings.Split(in, ";")) == 5 && re.MatchString(strings.Split(in, ";")[0]) {
		return true
	}

	return false
}

func ParseSessionString(in string) (email string, sess session.Session, err error) {
	if !isUnencryptedSession(in) {
		err = errors.New("session invalid, or encrypted and key was not provided")
		return
	}

	parts := strings.Split(in, ";")
	email = parts[0]
	sess = session.Session{
		Token:  parts[2],
		Server: parts[1],
	}

	return
}

func StringInSlice(inStr string, inSlice []string, matchCase bool) bool {
	for i := range inSlice {
		if matchCase && inStr == inSlice[i] {
			return true
		} else if strings.EqualFold(inStr, inSlice[i]) {
			return true
		}
	}

	return false
}

func stripTrailingSlash(in string) string {
	if strings.HasSuffix(in, "/") {
		return in[:len(in)-1]
	}

	return in
}

func colourDiff(diff string) string {
	switch diff {
	case identical:
		return green(diff)
	case localMissing:
		return red(diff)
	case localNewer:
		return yellow(diff)
	case untracked:
		return yellow(diff)
	case remoteNewer:
		return yellow(diff)
	default:
		return diff
	}
}

// binarySniffBytes is how much of a file is examined to decide whether it is
// text, matching the size git uses for the same decision.
const binarySniffBytes = 8000

// isBinaryFile reports whether a file's content cannot survive being stored as
// note text. A note is encoded as JSON, and encoding/json replaces invalid
// UTF-8 with U+FFFD rather than failing, so pushing such a file would appear to
// succeed and return corrupted content on the way back. A NUL byte is the
// usual marker of a binary file and is treated the same way.
func isBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}

	defer func() {
		_ = f.Close()
	}()

	buf := make([]byte, binarySniffBytes)

	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	buf = buf[:n]

	if bytes.IndexByte(buf, 0) >= 0 {
		return true, nil
	}

	// A rune can straddle the end of the window, so drop a trailing partial one
	// rather than reporting the file binary because it was cut mid-character.
	for len(buf) > 0 && !utf8.Valid(buf) {
		r, size := utf8.DecodeLastRune(buf)
		if r != utf8.RuneError || size != 1 || len(buf) < binarySniffBytes {
			break
		}

		buf = buf[:len(buf)-1]
	}

	return !utf8.Valid(buf), nil
}

// editorArea is the component area Standard Notes uses for editors.
const editorArea = "editor-editor"

// EditorAssociation is a tracked note that a Standard Notes editor has been
// associated with.
type EditorAssociation struct {
	// NotePath is the note's path relative to home, for example .gitconfig
	NotePath string
	// Editor is the editor's name, falling back to its identifier
	Editor string
}

// findEditorAssociations reports tracked notes that an editor component claims.
// Dotfiles are plain text, and an editor that stores anything else - Super
// keeps JSON, Rich Text keeps HTML - rewrites the note's content when it saves,
// so the file pulled back is not the file that was pushed.
func findEditorAssociations(items gosn.Items, twn tagsWithNotes, home, rootTag string) []EditorAssociation {
	rootTag = normalizeRootTag(rootTag)

	// the notes sn-dotfiles tracks, by uuid
	tracked := make(map[string]string)

	for _, t := range twn {
		for _, note := range t.notes {
			path, err := tagTitleToFSDir(t.tag.Content.GetTitle(), home, rootTag)
			if err != nil {
				continue
			}

			tracked[note.GetUUID()] = stripHome(path+note.Content.GetTitle(), home)
		}
	}

	var found []EditorAssociation

	for _, item := range items {
		if item.GetContentType() != common.SNItemTypeComponent || item.GetContent() == nil {
			continue
		}

		content, ok := item.GetContent().(*gosn.ComponentContent)
		if !ok || content.Area != editorArea {
			continue
		}

		editor := content.Name
		if editor == "" {
			editor = "an editor"
		}

		for _, uuid := range content.GetItemAssociations() {
			if path, isTracked := tracked[uuid]; isTracked {
				found = append(found, EditorAssociation{NotePath: path, Editor: editor})
			}
		}
	}

	sort.Slice(found, func(i, j int) bool { return found[i].NotePath < found[j].NotePath })

	return found
}

// editorAssociationWarning renders the warning appended to status and sync
// output, and is empty when nothing is associated.
func editorAssociationWarning(associations []EditorAssociation) string {
	if len(associations) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("\n\n")
	sb.WriteString(yellow("warning: an editor is associated with tracked dotfiles"))
	sb.WriteString("\n")

	for _, a := range associations {
		fmt.Fprintf(&sb, "  %s | %s\n", bold(addDot(a.NotePath)), a.Editor)
	}

	sb.WriteString("\nDotfiles are plain text. An editor that stores anything else rewrites the\n")
	sb.WriteString("note when it saves, and the file pulled back will not be the file pushed.\n")
	sb.WriteString("Remove the association in the Standard Notes app, or set the note to use\n")
	sb.WriteString("the plain text editor.\n")

	return sb.String()
}
