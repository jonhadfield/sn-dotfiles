package sndotfiles

import (
	"os"
	"strings"
)

// Most tests in this package talk to a live Standard Notes account: they sign
// in, then create and delete real notes and tags. They are opt-in, so set
// SN_INTEGRATION_TESTS=1 to run them. Without it only the offline unit tests
// run, which means `go test ./...` works on a machine with no credentials.
func integrationEnabled() bool {
	switch strings.ToLower(os.Getenv("SN_INTEGRATION_TESTS")) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
