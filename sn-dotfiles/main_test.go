package sndotfiles

import (
	"os"
	"testing"

	"github.com/jonhadfield/dotfiles-sn/internal/snmock"
	"github.com/jonhadfield/gosn-v2/cache"
)

var (
	testCacheSession *cache.Session
	// testMockServer is the mock account backing the tests, and is nil when
	// they are running against a real Standard Notes account.
	testMockServer *snmock.Server
)

// requireLiveSession skips the calling test when there is no session to run
// against. TestMain falls back to the mock server when no real account is
// configured, so this only bites if that setup failed.
func requireLiveSession(t *testing.T) {
	t.Helper()

	if testCacheSession == nil {
		t.Skip("skipping: no Standard Notes session available")
	}
}

// requireMockServer skips the calling test when it is not running against the
// mock server, for assertions that need to inspect what the server holds.
func requireMockServer(t *testing.T) *snmock.Server {
	t.Helper()

	requireLiveSession(t)

	if testMockServer == nil {
		t.Skip("skipping: test inspects mock server state and a real account is configured")
	}

	return testMockServer
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	sess, srv, cleanup, err := snmock.NewSession(SNAppName, true)
	if err != nil {
		panic(err)
	}

	defer cleanup()

	testCacheSession = sess
	testMockServer = srv

	return m.Run()
}
