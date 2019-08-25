package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	sndotfiles "github.com/jonhadfield/dotfiles-sn"
	"github.com/spf13/viper"
	keyring "github.com/zalando/go-keyring"

	"github.com/jonhadfield/gosn"

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

func TestStripHome(t *testing.T) {
	res, err := stripHome("/home/bob/something/else.txt", "/home/bob")
	assert.NoError(t, err)
	assert.Equal(t, "something/else.txt", res)
	res, err = stripHome("/home/bob/something/else.txt", "")
	assert.Error(t, err)
	assert.Empty(t, res)
	res, err = stripHome("", "/home/bob")
	assert.Error(t, err)
	assert.Empty(t, res)
}

func TestIsValidDotfilePath(t *testing.T) {
	home := getHome()
	assert.True(t, isValidDotfilePath(fmt.Sprintf("%s/.test", home)))
	assert.True(t, isValidDotfilePath(fmt.Sprintf("%s/.test/file.txt", home)))
	assert.True(t, isValidDotfilePath(fmt.Sprintf("%s/.test/test2/file.txt", home)))
	assert.False(t, isValidDotfilePath(fmt.Sprintf("%s/test/test2/file.txt", home)))
	assert.False(t, isValidDotfilePath(fmt.Sprintf("%s/test", home)))
}

func TestAdd(t *testing.T) {
	viper.SetEnvPrefix("sn")
	assert.NoError(t, viper.BindEnv("email"))
	assert.NoError(t, viper.BindEnv("password"))
	assert.NoError(t, viper.BindEnv("server"))
	serverURL := os.Getenv("SN_SERVER")
	if serverURL == "" {
		serverURL = sndotfiles.SNServerURL
	}
	session, _, err := sndotfiles.GetSession(false, serverURL)
	defer func() {
		if _, err := sndotfiles.WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	home := getHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	msg, disp, err := startCLI([]string{"sn-dotfiles", "add", applePath})
	assert.NotEmpty(t, msg)
	assert.True(t, disp)
	assert.Contains(t, msg, "2 tags")
	assert.Contains(t, msg, "1 files")
	assert.NoError(t, err)

}

func TestWipe(t *testing.T) {
	viper.SetEnvPrefix("sn")
	assert.NoError(t, viper.BindEnv("email"))
	assert.NoError(t, viper.BindEnv("password"))
	assert.NoError(t, viper.BindEnv("server"))

	home := getHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	serverURL := os.Getenv("SN_SERVER")
	if serverURL == "" {
		serverURL = sndotfiles.SNServerURL
	}
	session, _, err := sndotfiles.GetSession(false, serverURL)
	assert.NoError(t, err)
	_, _, _, _, _, _, err = sndotfiles.Add(session, home, []string{applePath}, true)
	msg, disp, err := startCLI([]string{"sn-dotfiles", "wipe", "--force"})
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)
	assert.Contains(t, msg, "3 ")
	assert.True(t, disp)
}

func TestAddSession(t *testing.T) {
	viper.SetEnvPrefix("sn")
	assert.NoError(t, viper.BindEnv("email"))
	assert.NoError(t, viper.BindEnv("password"))
	assert.NoError(t, viper.BindEnv("server"))
	serverURL := os.Getenv("SN_SERVER")
	if serverURL == "" {
		serverURL = sndotfiles.SNServerURL
	}
	res := addSession(serverURL)
	assert.Contains(t, res, "successfully")
}

func TestNumTrue(t *testing.T) {
	assert.Equal(t, 3, numTrue(true, false, true, true))
	assert.Equal(t, 0, numTrue())
}

func createPathWithContent(path, content string) error {
	dir, _ := filepath.Split(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return f.Close()
}
func createTemporaryFiles(fwc map[string]string) error {
	for f, c := range fwc {
		if err := createPathWithContent(f, c); err != nil {
			return err
		}
	}
	return nil
}
