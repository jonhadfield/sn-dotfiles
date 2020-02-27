package sndotfiles

import (
	"github.com/jonhadfield/gosn-v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWipeInvalidSession(t *testing.T) {
	n, err := WipeDotfileTagsAndNotes(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, DefaultPageSize, true)
	assert.Zero(t, n)
	assert.Error(t, err)
}


func TestWipeNoItems(t *testing.T) {
	session, err := GetTestSession()
	assert.NoError(t, err)
	assert.NotEmpty(t, session.Token)
	var num int
	num, err = WipeDotfileTagsAndNotes(session, DefaultPageSize, true)
	assert.NoError(t, err)
	assert.Equal(t, 0, num)
}
