package sndotfiles

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/fatih/color"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/items"
)

const (
	// SNServerURL defines the default URL for making calls to sync with SN
	SNServerURL = common.APIServer
	// DotFilesTag defines the default tag that all SN Dotfiles will be prefixed with
	DotFilesTag = "dotfiles"
	// DefaultPageSize defines the number of items to attempt to sync per request
	DefaultPageSize = common.PageSize
	// SpinnerCharSet defines the characters to use for the spinner shown when syncing
	SpinnerCharSet = 14
	// SpinnerDelay defines the number of milliseconds to wait between each character in the spinner
	SpinnerDelay = 100

	SNAppName = "sn-dotfiles"
)

var (
	bold   = color.New(color.Bold).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// dotFilesTagRegexp matches the dotfiles tag itself and any of its descendants,
// e.g. "dotfiles" and "dotfiles.config.nvim", but not "mydotfiles".
var dotFilesTagRegexp = regexp.MustCompile(fmt.Sprintf(`^%s(\..+)?$`, regexp.QuoteMeta(DotFilesTag)))

func getTagsWithNotes(db *storm.DB, sess *cache.Session) (t tagsWithNotes, err error) {
	// validate session
	if !sess.Valid() {
		err = errors.New("invalid session")
		return
	}

	var notesAndTags cache.Items

	contentTypes := []string{
		common.SNItemTypeNote,
		common.SNItemTypeTag,
		common.SNItemTypeComponent,
		common.SNItemTypeExtension,
	}

	if e := db.Select(q.In("ContentType", contentTypes)).Find(&notesAndTags); e != nil {
		if !errors.Is(e, storm.ErrNotFound) {
			return t, e
		}
	}

	var parsed items.Items

	parsed, err = notesAndTags.ToItems(sess)
	if err != nil {
		return
	}

	var dotfileTags items.Tags

	var notes items.Notes

	for _, item := range parsed {
		if item.GetContent() == nil {
			continue
		}

		switch item.GetContentType() {
		case common.SNItemTypeTag:
			tag, ok := item.(*items.Tag)
			if ok && dotFilesTagRegexp.MatchString(tag.Content.GetTitle()) {
				dotfileTags = append(dotfileTags, *tag)
			}
		case common.SNItemTypeNote:
			if note, ok := item.(*items.Note); ok {
				notes = append(notes, *note)
			}
		}
	}

	for _, dotfileTag := range dotfileTags {
		twn := tagWithNotes{
			tag: dotfileTag,
		}

		noteRefIDs := getItemNoteRefIds(dotfileTag.Content.References())

		for _, note := range notes {
			if StringInSlice(note.GetUUID(), noteRefIDs, true) {
				twn.notes = append(twn.notes, note)
			}
		}

		t = append(t, twn)
	}

	return t, err
}

func getItemNoteRefIds(itemRefs items.ItemReferences) (refIds []string) {
	for _, ir := range itemRefs {
		if ir.ContentType == common.SNItemTypeNote {
			refIds = append(refIds, ir.UUID)
		}
	}

	return refIds
}

type tagWithNotes struct {
	tag   items.Tag
	notes items.Notes
}

type tagsWithNotes []tagWithNotes
