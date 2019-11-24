package sndotfiles

import (
	"fmt"

	"github.com/jonhadfield/gosn"
)

func WipeDotfileTagsAndNotes(session gosn.Session, debug bool) (int, error) {
	twns, err := get(session, debug)
	if err != nil {
		return 0, err
	}

	var itemsToRemove gosn.Items

	for _, twn := range twns {
		twn.tag.Deleted = true
		itemsToRemove = append(itemsToRemove, twn.tag)

		for _, n := range twn.notes {
			n.Deleted = true
			itemsToRemove = append(itemsToRemove, n)
		}
	}

	debugPrint(debug, fmt.Sprintf("WipeDotfileTagsAndNotes | removing %d items", len(itemsToRemove)))

	var eItemsToDel gosn.EncryptedItems

	if len(itemsToRemove) == 0 {
		return 0, nil
	}

	eItemsToDel, err = itemsToRemove.Encrypt(session.Mk, session.Ak, debug)
	if err != nil {
		return 0, err
	}

	putItemsInput := gosn.PutItemsInput{
		Session: session,
		Items:   eItemsToDel,
		Debug:   true,
	}

	_, err = gosn.PutItems(putItemsInput)
	if err != nil {
		return 0, err
	}

	return len(itemsToRemove), err
}
