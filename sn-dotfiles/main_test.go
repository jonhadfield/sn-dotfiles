package sndotfiles

import (
	"fmt"

	"github.com/jonhadfield/dotfiles-sn/internal/snmock"
	"github.com/jonhadfield/gosn-v2/auth"

	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/session"
	"github.com/spf13/viper"
	"os"
	"testing"
)

var (
	testCacheSession *cache.Session
	// testMockServer backs the tests when they are not running against a real
	// Standard Notes account, and is nil when they are.
	testMockServer *snmock.Server
)

func TestMain(m *testing.M) {
	// Without SN_INTEGRATION_TESTS, run against an in-memory server rather
	// than a real account. Only the server is fake: sign-in, key derivation,
	// encryption and the sync round trip are the real ones.
	if !integrationEnabled() {
		os.Exit(runAgainstMock(m))
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

	path, err = cache.GenCacheDBPath(*testCacheSession, "", SNAppName)
	if err != nil {
		panic(err)
	}

	testCacheSession.CacheDBPath = path
	os.Exit(m.Run())
}

// runAgainstMock points the tests at a mock Standard Notes server. It returns
// the exit code rather than calling os.Exit so that the server's cleanup runs.
func runAgainstMock(m *testing.M) int {
	srv, err := snmock.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to start the mock Standard Notes server:", err)

		return 1
	}

	defer srv.Close()

	testMockServer = srv

	in, err := auth.CliSignIn(snmock.Email, snmock.Password, srv.URL, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to sign in to the mock server:", err)

		return 1
	}

	testCacheSession = &cache.Session{
		Session: &session.Session{
			Debug:             true,
			HTTPClient:        common.NewHTTPClient(),
			Server:            srv.URL,
			MasterKey:         in.MasterKey,
			KeyParams:         in.KeyParams,
			AccessToken:       in.AccessToken,
			RefreshToken:      in.RefreshToken,
			AccessExpiration:  in.AccessExpiration,
			RefreshExpiration: in.RefreshExpiration,
		},
	}

	path, err := cache.GenCacheDBPath(*testCacheSession, os.TempDir(), SNAppName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create the cache db path:", err)

		return 1
	}

	testCacheSession.CacheDBPath = path

	return m.Run()
}

// requireLiveSession skips the calling test when there is no session to run
// against. TestMain provides the mock server when no real account is
// configured, so this only bites if that setup failed.
func requireLiveSession(t *testing.T) {
	t.Helper()

	if testCacheSession == nil {
		t.Skip("skipping: no Standard Notes session available")
	}
}

// requireMockServer returns the mock server backing the tests, skipping when
// they are running against a real account and there is none to inspect.
func requireMockServer(t *testing.T) *snmock.Server {
	t.Helper()

	requireLiveSession(t)

	if testMockServer == nil {
		t.Skip("skipping: test inspects mock server state and a real account is configured")
	}

	return testMockServer
}
