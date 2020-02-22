package sndotfiles

import (
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"regexp"
	"time"

	"github.com/fatih/color"
)

const (
	// SNServerURL defines the default URL for making calls to sync with SN
	SNServerURL = "https://sync.standardnotes.org"
	// DotFilesTag defines the default tag that all SN Dotfiles will be prefixed with
	DotFilesTag = "dotfiles"
	// DefaultPageSize defines the number of items to attempt to sync per request
	DefaultPageSize = 500
)

var (
	bold   = color.New(color.Bold).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

func removeImposters(i gosn.EncryptedItems) (o gosn.EncryptedItems) {
	for _, x := range i {
		if StringInSlice(x.ContentType, []string{"Note", "Tag"}, false) {
			o = append(o, x)
		}
	}

	return o
}

func get(session gosn.Session, pageSize int, debug bool) (t tagsWithNotes, err error) {
	si := gosn.SyncInput{
		Session:  session,
		PageSize: pageSize,
		Debug:    debug,
	}

	var so gosn.SyncOutput

	start := time.Now()
	so, err = gosn.Sync(si)
	elapsed := time.Since(start)
	debugPrint(debug, fmt.Sprintf("get | get took: %v", elapsed))

	if err != nil {
		return t, err
	}

	so.Items = removeImposters(so.Items)
	so.Items.DeDupe()

	var dItems gosn.DecryptedItems

	dItems, err = so.Items.Decrypt(session.Mk, session.Ak, debug)
	if err != nil {
		return
	}

	var items gosn.Items

	items, err = dItems.Parse()

	if err != nil {
		return
	}

	var dotfileTags gosn.Tags

	var notes gosn.Notes

	r := regexp.MustCompile(fmt.Sprintf("%s.?.*", DotFilesTag))

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
	Session    gosn.Session
	Filters    gosn.ItemFilters
	NoteTitles []string
	TagTitles  []string
	TagUUIDs   []string
	PageSize   int
	BatchSize  int
	Debug      bool
}
