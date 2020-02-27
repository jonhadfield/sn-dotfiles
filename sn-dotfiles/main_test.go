package sndotfiles

import (
	"github.com/jonhadfield/gosn-v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	_, err := get(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	}, DefaultPageSize, true)
	assert.Error(t, err)
}
