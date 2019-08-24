package main

import (
	"fmt"
	"github.com/jonhadfield/gosn"
	"github.com/zalando/go-keyring"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCLIInvalidCommand(t *testing.T) {
	// Run the crashing code when FLAG is set
	if os.Getenv("FLAG") == "1" {
		msg, display, err := startCLI([]string{"sn-dotfiles", "lemon"})
		fmt.Println(msg, display, err)
		return
	}
	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestCLIInvalidCommand")
	cmd.Env = append(os.Environ(), "FLAG=1")
	err := cmd.Run()

	// Cast the error as *exec.ExitError and compare the result
	e, ok := err.(*exec.ExitError)
	expectedErrorString := "exit status 1"
	assert.Equal(t, true, ok)
	assert.Equal(t, expectedErrorString, e.Error())
}

var (
	testSessionEmail  = "me@home.com"
	testSessionServer = "https://sync.server.com"
	testSessionToken  = "testsessiontoken"
	testSessionAk     = "testsessionak"
	testSessionMk     = "testsessionmk"
	testSession       = fmt.Sprintf("%s;%s;%s;%s;%s", testSessionEmail, testSessionServer,
		testSessionToken, testSessionAk, testSessionMk)
)

func TestGetSession(t *testing.T) {
	keyring.MockInit()
	err := keyring.Set(service, "session", testSession)
	assert.NoError(t, err)

	var s, errMsg string
	s, errMsg = getSession()
	assert.Empty(t, errMsg)
	assert.Equal(t, testSession, s)
}

func TestRemoveSession(t *testing.T) {
	keyring.MockInit()
	err := keyring.Set(service, "session", testSession)
	assert.NoError(t, err)

	msg := removeSession()
	assert.Equal(t, msgSessionRemovalSuccess, msg)
	msg = removeSession()
	assert.Equal(t, fmt.Sprintf("%s: %s", msgSessionRemovalFailure, "secret not found in keyring"), msg)
}

func TestMakeSessionString(t *testing.T) {
	sess := gosn.Session{
		Token:  testSessionToken,
		Mk:     testSessionMk,
		Ak:     testSessionAk,
		Server: testSessionServer,
	}
	ss := makeSessionString(testSessionEmail, sess)
	assert.Equal(t, testSession, ss)
}

func TestParseSessionString(t *testing.T) {
	ss, err := parseSessionString(testSession)
	assert.NoError(t, err)
	assert.Len(t, ss, 5)
	assert.Equal(t, testSessionEmail, ss[0])
	assert.Equal(t, testSessionServer, ss[1])
	assert.Equal(t, testSessionToken, ss[2])
	assert.Equal(t, testSessionAk, ss[3])
	assert.Equal(t, testSessionMk, ss[4])
}