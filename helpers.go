package sndotfiles

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jonhadfield/gosn"
	"github.com/lithammer/shortuuid"
	"github.com/zalando/go-keyring"
)

func debugPrint(show bool, msg string) {
	if show {
		log.Println(msg)
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

func getTemporaryHome() string {
	home := fmt.Sprintf("%s/%s", os.TempDir(), shortuuid.New())
	return strings.ReplaceAll(home, "//", "/")
}

func stripHome(in, home string) string {
	if home != "" && strings.HasPrefix(in, home) {
		return in[len(home)+1:]
	}
	return in
}

func push(session gosn.Session, itemDiffs []ItemDiff) (pio gosn.PutItemsOutput, err error) {
	var dItems gosn.Items
	for _, i := range itemDiffs {
		dItems = append(dItems, i.remote)
	}
	if dItems == nil {
		err = errors.New("no items to push")
		return
	}

	return putItems(session, dItems)
}

func getTagIfExists(name string, twn tagsWithNotes) (tag gosn.Item, found bool) {
	for _, x := range twn {
		if name == x.tag.Content.GetTitle() {
			return x.tag, true
		}
	}
	return tag, false
}

func createMissingTags(session gosn.Session, pt string, twn tagsWithNotes) (newTags gosn.Items, err error) {
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
			itemsToPush = append(itemsToPush, createTag(f))
		}
	}

	var pio gosn.PutItemsOutput
	pio, err = putItems(session, itemsToPush)
	if err != nil {
		return
	}
	created := pio.ResponseBody.SavedItems
	created.DeDupe()
	return created.DecryptAndParse(session.Mk, session.Ak)
}

func pushAndTag(session gosn.Session, tim map[string]gosn.Items, twn tagsWithNotes) (tagsPushed, notesPushed int, err error) {
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
					UUID:        note.UUID,
					ContentType: "Note",
				})
			}
			existingTag.Content.UpsertReferences(newReferences)
			itemsToPush = append(itemsToPush, existingTag)
		} else {
			// need to create tag
			var newTags gosn.Items
			newTags, err = createMissingTags(session, potentialTag, twn)
			if err != nil {
				return
			}
			// create a new item reference for each note to be tagged
			var newReferences gosn.ItemReferences
			for _, note := range notes {
				itemsToPush = append(itemsToPush, note)
				newReferences = append(newReferences, gosn.ItemReference{
					UUID:        note.UUID,
					ContentType: "Note",
				})
			}
			newTag := newTags[len(newTags)-1]
			newTag.Content.UpsertReferences(newReferences)
			itemsToPush = append(itemsToPush, newTag)

			// add to twn so we don't get duplicates
			twn = append(twn, tagWithNotes{
				tag:   newTag,
				notes: notes,
			})
			for x := 0; x < len(newTags)-1; x++ {
				twn = append(twn, tagWithNotes{
					tag:   newTags[x],
					notes: nil,
				})
			}
		}
	}

	_, err = putItems(session, itemsToPush)
	tagsPushed, notesPushed = getItemCounts(itemsToPush)
	return
}

func getItemCounts(items gosn.Items) (tags, notes int) {
	for _, item := range items {
		if item.ContentType == "Note" {
			notes++
		}
		if item.ContentType == "Tag" {
			tags++
		}
	}
	return
}

func createTag(name string) (tag gosn.Item) {
	dfTagContent := gosn.NewTagContent()
	tag = *gosn.NewTag()
	dfTagContent.Title = name
	tag.Content = dfTagContent
	tag.UUID = gosn.GenUUID()
	return
}

func pull(itemDiffs []ItemDiff) error {
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

func itemInItems(item gosn.Item, items gosn.Items) bool {
	for _, i := range items {
		if i.UUID == item.UUID {
			return true
		}
	}
	return false
}

func getAllTagsWithoutNotes(twn tagsWithNotes, deletedNotes gosn.Items) (tagsWithoutNotes []string) {
	// get a map of all tags and notes, minus the notes to delete
	res := make(map[string]int)
	// initialise map with 0 count
	for _, x := range twn {
		res[x.tag.Content.GetTitle()] = 0
	}
	// get a count of notes for each tag
	for _, t := range twn {
		// generate list of tags to reduce later
		for _, n := range t.notes {
			if !itemInItems(n, deletedNotes) {
				res[t.tag.Content.GetTitle()]++
			}
		}
	}
	// create list of tags without notes
	for tn, count := range res {
		if count == 0 {
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

func findEmptyTags(twn tagsWithNotes, deletedNotes gosn.Items, debug bool) gosn.Items {
	allTagsWithoutNotes := getAllTagsWithoutNotes(twn, deletedNotes)

	// generate a map of tag child counts
	allTagsChildMap := make(map[string][]string)

	var tagsToRemove []string
	var allDotfileChildTags []string
	// for each tag, the last item is the child
	for _, atwn := range twn {
		if strings.HasPrefix(atwn.tag.Content.GetTitle(), DotFilesTag+".") {
			allDotfileChildTags = append(allDotfileChildTags, atwn.tag.Content.GetTitle())
		}
		tagTitle := atwn.tag.Content.GetTitle()
		splitTag := strings.Split(tagTitle, ".")
		if strings.Contains(tagTitle, ".") {
			firstPart := splitTag[:len(splitTag)-1]
			lastPart := splitTag[len(splitTag)-1:]
			allTagsChildMap[strings.Join(firstPart, ".")] = append(allTagsChildMap[strings.Join(firstPart, ".")], strings.Join(lastPart, "."))
		}
	}

	// remove tags without notes and without children
	for {
		var changeMade bool
		for k, v := range allTagsChildMap {
			for _, i := range v {
				completeTag := k + "." + i
				// check if noteless tag exists
				if StringInSlice(completeTag, allTagsWithoutNotes, true) {
					// check if tag still has children
					if len(allTagsChildMap[completeTag]) == 0 {
						allTagsChildMap[k] = removeStringFromSlice(i, v)
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
	// now remove tags without children
	for k, v := range allTagsChildMap {
		if len(v) == 0 {
			delete(allTagsChildMap, k)
			tagsToRemove = append(tagsToRemove, k)
		}
	}

	tagsToRemove = dedupe(tagsToRemove)

	// now remove dotfiles tag if it has no children
	if len(tagsToRemove) == len(allDotfileChildTags) {
		tagsToRemove = append(tagsToRemove, DotFilesTag)
		debugPrint(debug, fmt.Sprintf("findEmptyTags | removing '%s' tag as all children being removed", DotFilesTag))
	}
	debugPrint(debug, fmt.Sprintf("findEmptyTags | total to remove: %d", len(tagsToRemove)))
	return tagTitlesToTags(tagsToRemove, twn)
}

func tagTitlesToTags(tagTitles []string, twn tagsWithNotes) (res gosn.Items) {
	for _, t := range twn {
		if StringInSlice(t.tag.Content.GetTitle(), tagTitles, true) {
			res = append(res, t.tag)
		}
	}
	return
}

func getItemsToRemove(path, home string, twn tagsWithNotes) (homeRelPath string, res gosn.Items) {
	pathType, err := getPathType(path)
	if err != nil {
		return
	}
	var isDir bool
	if pathType == "dir" {
		isDir = true
	}
	homeRelPath = stripHome(path, home)

	remoteEquiv := homeRelPath

	// get item tags from remoteEquiv by stripping <DotFilesTag> and filename from remoteEquiv
	var noteTag, noteTitle string
	if !isDir {
		// split between tag and title if remote equivalent doesn't contain slash
		if strings.Contains(remoteEquiv, string(os.PathSeparator)) {
			remoteEquiv = stripDot(remoteEquiv)
			noteTag, noteTitle = filepath.Split(remoteEquiv)
			noteTag = DotFilesTag + "." + strings.ReplaceAll(noteTag[:len(noteTag)-1], string(os.PathSeparator), ".")
		} else {
			noteTag = DotFilesTag
			noteTitle = remoteEquiv
		}
		for _, t := range twn {
			if t.tag.Content.GetTitle() == noteTag {
				for _, note := range t.notes {
					if note.Content.GetTitle() == noteTitle {
						res = append(res, note)
					}
				}
			}
		}
	} else {
		// tag specified so find all notes matching tag and tags underneath
		remoteEquiv = stripDot(remoteEquiv)

		// strip trailing slash if provided
		if strings.HasSuffix(remoteEquiv, string(os.PathSeparator)) {
			remoteEquiv = remoteEquiv[:len(remoteEquiv)-1]
		}
		noteTag = DotFilesTag + "." + remoteEquiv
		// find notes matching tag
		for _, t := range twn {
			if t.tag.Content.GetTitle() == noteTag || strings.HasPrefix(t.tag.Content.GetTitle(), noteTag+".") {
				for _, note := range t.notes {
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
	return homeRelPath, res
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

func isSymlink(path string) (res bool, err error) {
	var f os.FileInfo
	f, err = os.Lstat(path)
	if err != nil {
		return
	}
	return f.Mode()&os.ModeSymlink != 0, err
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

func tagTitleToFSDIR(title, home string) (path string, isHome bool, err error) {
	if title == "" {
		err = errors.New("tag title required")
		return
	}
	if home == "" {
		err = errors.New("Home directory required")
		return
	}
	if !strings.HasPrefix(title, DotFilesTag) {
		return
	}
	if title == DotFilesTag {
		return home + string(os.PathSeparator), true, nil
	}
	a := title[len(DotFilesTag)+1:]
	b := strings.ReplaceAll(a, ".", string(os.PathSeparator))
	c := addDot(b)
	return home + string(os.PathSeparator) + c + string(os.PathSeparator), false, err
}

func pathToTag(homeRelPath string) string {
	// prepend dotfiles path
	r := DotFilesTag + homeRelPath
	// replace path separators with dots
	r = strings.ReplaceAll(r, string(os.PathSeparator), ".")
	if strings.HasSuffix(r, ".") {
		return r[:len(r)-1]
	}
	return r
}
func GetSession(loadSession bool, server string) (session gosn.Session, email string, err error) {
	if loadSession {
		service := "StandardNotesCLI"
		var rawSess string
		rawSess, err = keyring.Get(service, "Session")
		if err != nil {
			return
		}
		email, session, err = ParseSessionString(rawSess)
		if err != nil {
			return
		}
	} else {
		session, email, err = GetSessionFromUser(server)
		if err != nil {
			return
		}
	}
	return
}
func GetSessionFromUser(server string) (gosn.Session, string, error) {
	var sess gosn.Session
	var email string
	var err error
	var password, apiServer, errMsg string
	email, password, apiServer, errMsg = GetCredentials(server)
	if errMsg != "" {
		fmt.Printf("\nerror: %s\n\n", errMsg)
		return sess, email, err
	}
	sess, err = gosn.CliSignIn(email, password, apiServer)
	if err != nil {
		return sess, email, err

	}
	return sess, email, err
}

func ParseSessionString(in string) (email string, session gosn.Session, err error) {
	parts := strings.Split(in, ";")
	if len(parts) != 5 {
		err = errors.New("invalid Session found")
		return
	}
	email = parts[0]
	session = gosn.Session{
		Token:  parts[2],
		Mk:     parts[4],
		Ak:     parts[3],
		Server: parts[1],
	}
	return
}
func StringInSlice(inStr string, inSlice []string, matchCase bool) bool {
	for i := range inSlice {
		if matchCase && inStr == inSlice[i] {
			return true
		} else if strings.ToLower(inStr) == strings.ToLower(inSlice[i]) {
			return true
		}
	}
	return false
}

func putItems(session gosn.Session, items gosn.Items) (pio gosn.PutItemsOutput, err error) {
	var encItemsToPut gosn.EncryptedItems
	encItemsToPut, err = items.Encrypt(session.Mk, session.Ak)
	if err != nil {
		return pio, fmt.Errorf("failed to encrypt items to put: %v", err)
	}
	pii := gosn.PutItemsInput{
		Items:   encItemsToPut,
		Session: session,
	}
	return gosn.PutItems(pii)
}
