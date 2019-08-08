package sndotfiles

import (
	"os"

	"github.com/jonhadfield/gosn"
)

func getSession() (gosn.Session, error) {
	email := os.Getenv("SN_EMAIL")
	password := os.Getenv("SN_PASSWORD")
	apiServer := os.Getenv("SN_SERVER")
	return CliSignIn(email, password, apiServer)
}

func wipe(session gosn.Session) (int, error) {
	getItemsInput := gosn.GetItemsInput{
		Session: session,
	}
	var err error
	// get all existing Tags and Notes and mark for deletion
	var output gosn.GetItemsOutput
	output, err = gosn.GetItems(getItemsInput)
	if err != nil {
		return 0, err
	}
	output.Items.DeDupe()
	var pi gosn.Items
	pi, err = output.Items.DecryptAndParse(session.Mk, session.Ak)
	if err != nil {
		return 0, err
	}
	var itemsToDel gosn.Items
	for _, item := range pi {
		if item.Deleted {
			continue
		}

		switch {
		case item.ContentType == "Tag":
			item.Deleted = true
			item.Content = gosn.NewTagContent()
			itemsToDel = append(itemsToDel, item)
		case item.ContentType == "Note":
			item.Deleted = true
			item.Content = gosn.NewNoteContent()
			itemsToDel = append(itemsToDel, item)
		}
	}
	// delete items
	var eItemsToDel gosn.EncryptedItems
	eItemsToDel, err = itemsToDel.Encrypt(session.Mk, session.Ak)
	if err != nil {
		return 0, err
	}
	putItemsInput := gosn.PutItemsInput{
		Session:   session,
		Items:     eItemsToDel,
		SyncToken: output.SyncToken,
	}
	_, err = gosn.PutItems(putItemsInput)
	if err != nil {
		return 0, err
	}
	return len(itemsToDel), err
}
