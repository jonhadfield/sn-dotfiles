package main

import (
	"fmt"
	sndotfiles2 "github.com/jonhadfield/dotfiles-sn/sn-dotfiles"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	gosn "github.com/jonhadfield/gosn-v2/items"
	"github.com/jonhadfield/gosn-v2/session"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"index/suffixarray"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func removeDB(dbPath string) {
	if err := os.Remove(dbPath); err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			panic(err)
		}
	}
}

func CleanUp(session cache.Session) error {
	removeDB(session.CacheDBPath)
	_, err := gosn.DeleteContent(testCacheSession.Session, false)
	return err
}

var testCacheSession *cache.Session

func csync(si cache.SyncInput) (so cache.SyncOutput, err error) {
	return cache.Sync(cache.SyncInput{
		Session: si.Session,
		Close:   si.Close,
	})
}

func TestMain(m *testing.M) {
	// commands require a config file, so provide one that includes everything
	configHome, err := os.MkdirTemp("", "sn-dotfiles-config")
	if err != nil {
		panic(err)
	}

	if err = os.MkdirAll(filepath.Join(configHome, sndotfiles2.SNAppName), 0o700); err != nil {
		panic(err)
	}

	if err = os.WriteFile(filepath.Join(configHome, sndotfiles2.SNAppName, "config.yaml"), []byte("include:\n  - '.*'\n"), 0o600); err != nil {
		panic(err)
	}

	if err = os.Setenv("XDG_CONFIG_HOME", configHome); err != nil {
		panic(err)
	}

	// sign in the same way the CLI does, using SN_EMAIL, SN_PASSWORD and SN_SERVER
	viper.SetEnvPrefix("sn")
	_ = viper.BindEnv("email")
	_ = viper.BindEnv("password")

	sess, _, err := session.GetSession(common.NewHTTPClient(), false, "", os.Getenv("SN_SERVER"), true)
	if err != nil {
		panic(err)
	}

	testCacheSession = &cache.Session{Session: &sess}

	var path string

	path, err = cache.GenCacheDBPath(*testCacheSession, "", sndotfiles2.SNAppName)
	if err != nil {
		panic(err)
	}

	testCacheSession.CacheDBPath = path

	var so cache.SyncOutput
	so, err = csync(cache.SyncInput{
		Session: testCacheSession,
		Close:   false,
	})
	if err != nil {
		panic(err)
	}

	var allPersistedItems cache.Items

	if err = so.DB.All(&allPersistedItems); err != nil {
		return
	}
	if err = so.DB.Close(); err != nil {
		panic(err)
	}

	if testCacheSession.DefaultItemsKey.ItemsKey == "" {
		panic("failed in TestMain due to empty default items key")
	}

	code := m.Run()

	_ = os.RemoveAll(configHome)

	os.Exit(code)
}

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
	assert.True(t, isValidDotfilePath(fmt.Sprintf("%s/.test", home), home))
	assert.True(t, isValidDotfilePath(fmt.Sprintf("%s/.test/file.txt", home), home))
	assert.True(t, isValidDotfilePath(fmt.Sprintf("%s/.test/test2/file.txt", home), home))
	assert.False(t, isValidDotfilePath(fmt.Sprintf("%s/test/test2/file.txt", home), home))
	assert.False(t, isValidDotfilePath(fmt.Sprintf("%s/test", home), home))
}

// TestIsValidDotfilePathHonoursHome checks paths are judged against the home
// the caller passes, which is what --home-dir sets, rather than the real one
func TestIsValidDotfilePathHonoursHome(t *testing.T) {
	otherHome := filepath.Join(os.TempDir(), "sn-dotfiles-home")

	// a dotfile under the given home is valid even though it is not under $HOME
	assert.True(t, isValidDotfilePath(filepath.Join(otherHome, ".zshrc"), otherHome))
	assert.True(t, isValidDotfilePath(filepath.Join(otherHome, ".config", "fish", "config.fish"), otherHome))

	// and a path outside that home is not, whatever the real home is
	assert.False(t, isValidDotfilePath(filepath.Join(getHome(), ".zshrc"), otherHome))
	assert.False(t, isValidDotfilePath(filepath.Join(otherHome, "notadotfile"), otherHome))
}

func TestAdd(t *testing.T) {
	viper.SetEnvPrefix("sn")
	assert.NoError(t, viper.BindEnv("email"))
	assert.NoError(t, viper.BindEnv("password"))
	assert.NoError(t, viper.BindEnv("server"))
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	var err error
	home := getHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	var msg string
	var disp bool
	msg, disp, err = startCLI([]string{"sn-dotfiles", "add", applePath})
	assert.NotEmpty(t, msg)
	assert.True(t, disp)
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(".fruit/apple\\s*now tracked"), msg)
}

func TestAddInvalidPath(t *testing.T) {
	msg, disp, err := startCLI([]string{"sn-dotfiles", "add", "/invalid"})
	assert.NotEmpty(t, msg)
	assert.True(t, disp)
	assert.Contains(t, msg, "invalid")
	assert.NoError(t, err)
}

func TestAddAllAndPath(t *testing.T) {
	msg, disp, err := startCLI([]string{"sn-dotfiles", "add", "--all", "/invalid"})
	assert.NotEmpty(t, msg)
	assert.True(t, disp)
	assert.Contains(t, msg, "error: specifying --all and paths does not make sense")
	assert.NoError(t, err)
}

func TestAddNoArgs(t *testing.T) {
	msg, disp, err := startCLI([]string{"sn-dotfiles", "add"})
	assert.NotEmpty(t, msg)
	assert.True(t, disp)
	assert.Contains(t, msg, "error: either specify paths to add or --all to add everything")
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

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	msg, disp, err := startCLI([]string{"sn-dotfiles", "add", fmt.Sprintf("%s/.fruit", home)})
	assert.NoError(t, err)
	msg, disp, err = startCLI([]string{"sn-dotfiles", "remove", fmt.Sprintf("%s/.fruit", home)})
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)
	assert.Regexp(t, regexp.MustCompile(".fruit/apple\\s*removed"), msg)
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
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	ai := sndotfiles2.AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath}}
	_, err := sndotfiles2.Add(ai, true)
	assert.NoError(t, err)
	var msg string
	var disp bool
	time.Sleep(time.Second * 1)
	msg, disp, err = startCLI([]string{"sn-dotfiles", "wipe", "--force"})
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
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	var err error
	ai := sndotfiles2.AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath}}
	_, err = sndotfiles2.Add(ai, true)
	assert.NoError(t, err)
	var msg string
	var disp bool
	msg, disp, err = startCLI([]string{"sn-dotfiles", "status", applePath})
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
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	var err error
	ai := sndotfiles2.AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath, lemonPath}}
	_, err = sndotfiles2.Add(ai, true)
	assert.NoError(t, err)
	var msg string
	var disp bool
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
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
	assert.Len(t, results, 1)
}

func TestDiff(t *testing.T) {
	viper.SetEnvPrefix("sn")
	assert.NoError(t, viper.BindEnv("email"))
	assert.NoError(t, viper.BindEnv("password"))
	assert.NoError(t, viper.BindEnv("server"))

	home := getHome()
	fwc := make(map[string]string)
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	fwc[applePath] = "apple content"
	assert.NoError(t, createTemporaryFiles(fwc))
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()
	ai := sndotfiles2.AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath}}
	_, err := sndotfiles2.Add(ai, true)
	assert.NoError(t, err)
	var msg string
	var disp bool
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "diff", applePath})
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)
	assert.Contains(t, msg, "no differences")
	assert.True(t, disp)
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "diff", "~/.does/not/exist"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
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
	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	ai := sndotfiles2.AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath}}
	_, err := sndotfiles2.Add(ai, true)
	assert.NoError(t, err)
	var msg string
	var disp bool
	msg, disp, err = startCLI([]string{"sn-dotfiles", "--debug", "sync", applePath})
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

func TestMissingConfig(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")

	for _, command := range []string{"status", "sync", "add", "remove", "diff", "wipe"} {
		t.Run(command, func(t *testing.T) {
			_, _, err := startCLI([]string{"sn-dotfiles", "--config", missing, command, getHome() + "/.gitconfig"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "config file "+missing+" not found")
		})
	}
}

func TestInvalidConfigPattern(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("include:\n  - '('\n"), 0o600))

	_, _, err := startCLI([]string{"sn-dotfiles", "--config", path, "status"})
	require.Error(t, err)
	require.Contains(t, err.Error(), `invalid include pattern "("`)
}

func TestStatusIncludeRegex(t *testing.T) {
	home := getHome()
	applePath := fmt.Sprintf("%s/.fruit/apple", home)
	lemonPath := fmt.Sprintf("%s/.fruit/lemon", home)
	require.NoError(t, createTemporaryFiles(map[string]string{
		applePath: "apple content",
		lemonPath: "lemon content",
	}))

	defer func() {
		if err := CleanUp(*testCacheSession); err != nil {
			fmt.Println("failed to wipe")
		}
	}()

	ai := sndotfiles2.AddInput{Session: testCacheSession, Home: home, Paths: []string{applePath, lemonPath}}
	_, err := sndotfiles2.Add(ai, true)
	require.NoError(t, err)

	msg, _, err := startCLI([]string{"sn-dotfiles", "--include-regex", `^\.fruit/apple$`, "status"})
	require.NoError(t, err)
	require.Contains(t, msg, ".fruit/apple")
	require.NotContains(t, msg, ".fruit/lemon")

	msg, _, err = startCLI([]string{"sn-dotfiles", "--exclude-regex", `/apple$`, "status"})
	require.NoError(t, err)
	require.NotContains(t, msg, ".fruit/apple")
	require.Contains(t, msg, ".fruit/lemon")
}
