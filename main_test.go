package sndotfiles

import (
	"github.com/jonhadfield/gosn"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGet(t *testing.T) {
	_, err := get(gosn.Session{
		Token:  "invalid",
		Mk:     "invalid",
		Ak:     "invalid",
		Server: "invalid",
	})
	assert.Error(t, err)
}
