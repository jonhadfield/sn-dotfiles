package sndotfiles

import (
	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWipeInvalidSession(t *testing.T) {
	n, err := WipeDotfileTagsAndNotes(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, true)
	assert.Zero(t, n)
	assert.Error(t, err)
}