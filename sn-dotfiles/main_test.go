package sndotfiles

import (
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/session"
	"github.com/spf13/viper"
	"os"
	"testing"
)

var testCacheSession *cache.Session

func TestMain(m *testing.M) {
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
