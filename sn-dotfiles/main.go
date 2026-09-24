package sndotfiles

import (
	"errors"
	"fmt"
	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/fatih/color"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	gosn "github.com/jonhadfield/gosn-v2/items"
	"regexp"
	"strings"
)

const (
	// DotFilesTag is the default root tag that all SN Dotfiles are prefixed with
	DotFilesTag = "dotfiles"
	// DefaultPageSize defines the number of items to attempt to syncDBwithFS per request
	DefaultPageSize = 500
	// SpinnerCharSet defines the characters to use for the spinner shown when syncing
	SpinnerCharSet = 14
	// SpinnerDelay defines the number of milliseconds to wait between each character in the spinner
	SpinnerDelay = 100

	SNAppName = "sn-dotfiles"

	maxDebugChars = 120 // number of characters to display when logging API response body
)

// ResolveRootTag returns rootTag when set, otherwise DotFilesTag.
// Root tags must not contain '.' because dots separate path segments in SN tag titles.
func ResolveRootTag(rootTag string) (string, error) {
	rootTag = strings.TrimSpace(rootTag)
	if rootTag == "" {
		return DotFilesTag, nil
	}

	if err := ValidateRootTag(rootTag); err != nil {
		return "", err
	}

	return rootTag, nil
}

// ValidateRootTag reports whether rootTag is a valid SN root tag name.
func ValidateRootTag(rootTag string) error {
	if strings.TrimSpace(rootTag) == "" {
		return errors.New("root tag must not be empty")
	}

	if strings.Contains(rootTag, ".") {
		return fmt.Errorf("root tag %q must not contain '.' (dots separate path segments in Standard Notes tags)", rootTag)
	}

	if strings.ContainsAny(rootTag, `/\`) {
		return fmt.Errorf("root tag %q must not contain path separators", rootTag)
	}

	return nil
}

// normalizeRootTag returns DotFilesTag when rootTag is empty; it does not validate.
// Callers that accept user input should use ResolveRootTag instead.
func normalizeRootTag(rootTag string) string {
	if rootTag == "" {
		return DotFilesTag
	}

	return rootTag
}

var (
	bold   = color.New(color.Bold).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

func getTagsWithNotes(db *storm.DB, session *cache.Session, rootTag string) (t tagsWithNotes, err error) {
	// validate session
	if !session.Valid() {
		err = errors.New("invalid session")
		return
	}

	rootTag = normalizeRootTag(rootTag)

	var notesAndTags cache.Items

	if e := db.Select(q.In("ContentType", []string{"Note", "Tag"})).Find(&notesAndTags); e != nil {
		if e.Error() != "not found" {
			return t, e
		}
	}

	var items gosn.Items
	items, err = notesAndTags.ToItems(session)
	if err != nil {
		return
	}

	var dotfileTags gosn.Tags

	var notes gosn.Notes

	r := regexp.MustCompile(fmt.Sprintf(`^%s(\..+)?$`, regexp.QuoteMeta(rootTag)))

	for _, item := range items {
		if item.GetContent() != nil && item.GetContentType() == "Tag" && r.MatchString(item.GetContent().(*gosn.TagContent).Title) {
			tt := item.(*gosn.Tag)
			dotfileTags = append(dotfileTags, *tt)
		}

		if item.GetContentType() == "Note" && item.GetContent() != nil {
			n := item.(*gosn.Note)
			notes = append(notes, *n)
		}
	}

	for _, dotfileTag := range dotfileTags {
		twn := tagWithNotes{
			tag: dotfileTag,
		}

		for _, note := range notes {
			if StringInSlice(note.GetUUID(), getItemNoteRefIds(dotfileTag.GetContent().References()), false) {
				twn.notes = append(twn.notes, note)
			}
		}

		t = append(t, twn)
	}

	return t, err
}

// ListRootTags returns candidate root tags found in the account: undotted tag titles that
// either have a child tag (title.*) or a note whose title starts with '.'.
func ListRootTags(session *cache.Session) ([]string, error) {
	if !session.Valid() {
		return nil, errors.New("invalid session")
	}

	csi := cache.SyncInput{
		Session:    session,
		Close:      false,
		AlwaysSync: true,
	}

	cso, err := cache.Sync(csi)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cso.DB.Close() }()

	var notesAndTags cache.Items
	if e := cso.DB.Select(q.In("ContentType", []string{"Note", "Tag"})).Find(&notesAndTags); e != nil {
		if e.Error() != "not found" {
			return nil, e
		}
	}

	items, err := notesAndTags.ToItems(session)
	if err != nil {
		return nil, err
	}

	return rootTagsFromItems(items), nil
}

// rootTagsFromItems picks the candidate root tags out of a set of items: undotted
// tag titles that either have a child tag (title.*) or a note whose title starts
// with '.'. Separated from ListRootTags so it can be tested without an account.
func rootTagsFromItems(items gosn.Items) []string {
	notesByUUID := make(map[string]gosn.Note)
	childPrefix := make(map[string]bool)

	for _, item := range items {
		if item.GetContent() == nil {
			continue
		}

		switch item.GetContentType() {
		case "Note":
			n := item.(*gosn.Note)
			notesByUUID[n.GetUUID()] = *n
		case "Tag":
			title := item.GetContent().(*gosn.TagContent).Title
			if i := strings.Index(title, "."); i > 0 {
				childPrefix[title[:i]] = true
			}
		}
	}

	var roots []string
	seen := make(map[string]bool)

	for _, item := range items {
		if item.GetContent() == nil || item.GetContentType() != "Tag" {
			continue
		}

		title := item.GetContent().(*gosn.TagContent).Title
		if strings.Contains(title, ".") || seen[title] {
			continue
		}

		hasDotfileNote := false
		tag := item.(*gosn.Tag)
		for _, refID := range getItemNoteRefIds(tag.GetContent().References()) {
			if n, ok := notesByUUID[refID]; ok && strings.HasPrefix(n.Content.GetTitle(), ".") {
				hasDotfileNote = true
				break
			}
		}

		if hasDotfileNote || childPrefix[title] {
			roots = append(roots, title)
			seen[title] = true
		}
	}

	return roots
}

func getItemNoteRefIds(itemRefs gosn.ItemReferences) (refIds []string) {
	for _, ir := range itemRefs {
		if ir.ContentType == "Note" {
			refIds = append(refIds, ir.UUID)
		}
	}

	return refIds
}

type tagWithNotes struct {
	tag   gosn.Tag
	notes gosn.Notes
}

type tagsWithNotes []tagWithNotes

// GetNoteConfig defines the input for getting notes from SN
type GetNoteConfig struct {
	Session    cache.Session
	Filters    gosn.ItemFilters
	NoteTitles []string
	TagTitles  []string
	TagUUIDs   []string
	PageSize   int
	Debug      bool
}

// getEditorAssociations reports tracked notes that an editor component claims,
// so that a sync or status can warn before an editor rewrites one.
func getEditorAssociations(db *storm.DB, session *cache.Session, twn tagsWithNotes, home, rootTag string) ([]EditorAssociation, error) {
	var components cache.Items

	if e := db.Select(q.In("ContentType", []string{common.SNItemTypeComponent})).Find(&components); e != nil {
		if e.Error() != "not found" {
			return nil, e
		}

		return nil, nil
	}

	items, err := components.ToItems(session)
	if err != nil {
		return nil, err
	}

	return findEditorAssociations(items, twn, home, rootTag), nil
}
