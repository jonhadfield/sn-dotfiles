package sndotfiles

import (
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWipeInvalidSession(t *testing.T) {
	n, err := WipeDotfileTagsAndNotes(&cache.Session{}, DefaultPageSize, true)
	assert.Zero(t, n)
	assert.Error(t, err)
}

func TestWipeNoItems(t *testing.T) {
	var num int
	var err error
	num, err = WipeDotfileTagsAndNotes(testCacheSession, DefaultPageSize, true)
	assert.NoError(t, err)
	assert.Equal(t, 0, num)
}
