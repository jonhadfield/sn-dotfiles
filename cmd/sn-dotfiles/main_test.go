package main

import (
	"fmt"
	"index/suffixarray"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	sndotfiles "github.com/jonhadfield/dotfiles-sn"
	"github.com/jonhadfield/gosn"
	"github.com/spf13/viper"
	keyring "github.com/zalando/go-keyring"

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

func TestRemoveSession(t *testing.T) {
	keyring.MockInit()
	err := keyring.Set(service, sndotfiles.KeyringApplicationName, testSession)
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
	assert.Contains(t, msg, "1")
	assert.NoError(t, err)
}

func TestAddInvalidPath(t *testing.T) {
	msg, disp, err := startCLI([]string{"sn-dotfiles", "add", "/invalid"})
	assert.NotEmpty(t, msg)
	assert.True(t, disp)
	assert.Contains(t, msg, "invalid")
	assert.NoError(t, err)
}

func TestRemove(t *testing.T) {
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
	defer func() {
		if _, err := sndotfiles.WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	assert.NoError(t, err)
	ai := sndotfiles.AddInput{Session: session, Home: home, Paths: []string{applePath}, Debug: true}
	_, err = sndotfiles.Add(ai)
	assert.NoError(t, err)
	msg, disp, err := startCLI([]string{"sn-dotfiles", "remove", applePath})
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)
	assert.Contains(t, msg, "1 ")
	assert.True(t, disp)
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
	defer func() {
		if _, err := sndotfiles.WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	assert.NoError(t, err)
	ai := sndotfiles.AddInput{Session: session, Home: home, Paths: []string{applePath}, Debug: true}
	_, err = sndotfiles.Add(ai)
	assert.NoError(t, err)
	msg, disp, err := startCLI([]string{"sn-dotfiles", "wipe", "--force"})
	assert.NoError(t, err)
	assert.Contains(t, msg, "3 ")
	assert.True(t, disp)
}

func TestStatus(t *testing.T) {
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
	defer func() {
		if _, err := sndotfiles.WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	assert.NoError(t, err)
	ai := sndotfiles.AddInput{Session: session, Home: home, Paths: []string{applePath}, Debug: true}
	_, err = sndotfiles.Add(ai)
	assert.NoError(t, err)
	msg, disp, err := startCLI([]string{"sn-dotfiles", "status", applePath})
	assert.NoError(t, err)
	assert.Contains(t, msg, ".fruit/apple  identical")
	assert.True(t, disp)
}

func TestSync(t *testing.T) {
	viper.SetEnvPrefix("sn")
	assert.NoError(t, viper.BindEnv("email"))
	assert.NoError(t, viper.BindEnv("password"))
	assert.NoError(t, viper.BindEnv("server"))

	home := getHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	lemonPath := fmt.Sprintf("%s/.fruit/lemon", home)
	fwc[lemonPath] = "lemon content"
	assert.NoError(t, createTemporaryFiles(fwc))
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
	assert.NoError(t, err)
	ai := sndotfiles.AddInput{Session: session, Home: home, Paths: []string{applePath, lemonPath}, Debug: true}
	_, err = sndotfiles.Add(ai)
	assert.NoError(t, err)
	msg, disp, err := startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
	assert.NoError(t, err)
	assert.Contains(t, msg, "nothing to do")
	assert.True(t, disp)
	// test push
	fwc[applePath] = "apple content updated"
	// add delay so local file is recognised as newer
	time.Sleep(1 * time.Second)
	assert.NoError(t, createTemporaryFiles(fwc))
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
	assert.NoError(t, err)
	assert.Contains(t, msg, "pushed")
	// test pull - specify unchanged path and expect no change
	err = os.Remove(lemonPath)
	assert.NoError(t, err)
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
	assert.NoError(t, err)
	assert.Contains(t, msg, "nothing to do")
	// test pull - specify changed path (updated content set to be older) and expect change
	assert.NoError(t, err)

	fwc[lemonPath] = "lemon content updated"
	assert.NoError(t, createTemporaryFiles(fwc))

	tenMinsAgo := time.Now().Add(-time.Minute * 10)
	err = os.Chtimes(lemonPath, tenMinsAgo, tenMinsAgo)
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "sync", lemonPath})
	assert.NoError(t, err)
	r := regexp.MustCompile("pulled")
	index := suffixarray.New([]byte(msg))
	results := index.FindAllIndex(r, -1)
	fmt.Println(len(results))
	assert.Len(t, results, 1)
}

func TestSyncExclude(t *testing.T) {
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
	defer func() {
		if _, err := sndotfiles.WipeDotfileTagsAndNotes(session, true); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	assert.NoError(t, err)
	ai := sndotfiles.AddInput{Session: session, Home: home, Paths: []string{applePath}, Debug: true}
	_, err = sndotfiles.Add(ai)
	assert.NoError(t, err)
	msg, disp, err := startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
	assert.NoError(t, err)
	assert.Contains(t, msg, "nothing to do")
	assert.True(t, disp)
	fwc[applePath] = "apple content updated"
	// add delay so local file is recognised as newer
	time.Sleep(1 * time.Second)
	assert.NoError(t, createTemporaryFiles(fwc))
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
	assert.NoError(t, err)
	assert.Contains(t, msg, "pushed")
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
	res, err := addSession(serverURL)
	assert.NoError(t, err)
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
