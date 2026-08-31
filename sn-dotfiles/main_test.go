package sndotfiles

import (
	"os"
	"testing"

	"github.com/jonhadfield/gosn-v2/auth"
	"github.com/jonhadfield/gosn-v2/cache"
	snsession "github.com/jonhadfield/gosn-v2/session"
)

var testCacheSession *cache.Session

// requireLiveSession skips the calling test when no Standard Notes credentials
// were supplied, so that the offline unit tests can still be run.
func requireLiveSession(t *testing.T) {
	t.Helper()

	if testCacheSession == nil {
		t.Skip("skipping: SN_EMAIL and SN_PASSWORD not set")
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("SN_EMAIL") == "" || os.Getenv("SN_PASSWORD") == "" {
		os.Exit(m.Run())
	}

	gs, err := auth.CliSignIn(os.Getenv("SN_EMAIL"), os.Getenv("SN_PASSWORD"), os.Getenv("SN_SERVER"), true)
	if err != nil {
		panic(err)
	}

	testCacheSession = &cache.Session{
		Session: &snsession.Session{
			Debug:             true,
			Server:            gs.Server,
			Token:             gs.Token,
			MasterKey:         gs.MasterKey,
			RefreshExpiration: gs.RefreshExpiration,
			RefreshToken:      gs.RefreshToken,
			AccessToken:       gs.AccessToken,
			AccessExpiration:  gs.AccessExpiration,
			KeyParams:         gs.KeyParams,
		},
		CacheDBPath: "",
	}

	var path string

	path, err = cache.GenCacheDBPath(*testCacheSession, "", SNAppName)
	if err != nil {
		panic(err)
	}

	testCacheSession.CacheDBPath = path

	os.Exit(m.Run())
}
