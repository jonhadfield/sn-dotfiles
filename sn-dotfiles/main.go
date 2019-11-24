package sndotfiles

import (
	"fmt"
	"regexp"
	"time"

	"github.com/fatih/color"

	"github.com/jonhadfield/gosn"
)

const (
	// SNServerURL defines the default URL for making calls to sync with SN
	SNServerURL = "https://sync.standardnotes.org"
	// DotFilesTag defines the default tag that all SN Dotfiles will be prefixed with
	DotFilesTag = "dotfiles"
)

var (
	bold   = color.New(color.Bold).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

func get(session gosn.Session, debug bool) (t tagsWithNotes, err error) {
	getItemsInput := gosn.GetItemsInput{
		Session:  session,
		PageSize: 100,
		Debug:    debug,
	}

	var output gosn.GetItemsOutput
	start := time.Now()
	output, err = gosn.GetItems(getItemsInput)
	elapsed := time.Since(start)
	debugPrint(debug, fmt.Sprintf("get | get took: %v", elapsed))

	if err != nil {
		return t, err
	}

	output.Items.DeDupe()

	var dItems gosn.DecryptedItems
	dItems, err = output.Items.Decrypt(session.Mk, session.Ak, debug)

	if err != nil {
		return
	}

	var items gosn.Items

	items, err = dItems.Parse()
	if err != nil {
		return
	}
	// get all dotfile Tags and notes

	var dotfileTags gosn.Items

	var notes gosn.Items

	r := regexp.MustCompile(fmt.Sprintf("%s.?.*", DotFilesTag))

	for _, item := range items {
		if item.ContentType == "Tag" && item.Content != nil && r.MatchString(item.Content.GetTitle()) {
			dotfileTags = append(dotfileTags, item)
		}

		if item.ContentType == "Note" && item.Content != nil {
			notes = append(notes, item)
		}
	}

	for _, dotfileTag := range dotfileTags {
		twn := tagWithNotes{
			tag: dotfileTag,
		}

		for _, note := range notes {
			if StringInSlice(note.UUID, getItemNoteRefIds(dotfileTag.Content.References()), false) {
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
	tag   gosn.Item
	notes gosn.Items
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
