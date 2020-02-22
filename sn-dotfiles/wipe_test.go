package sndotfiles

import (
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
