package sndotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartBusySpinnerStopIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "missing.db")

	// Missing cache path selects the "initializing" prefix; Stop must be safe
	// to call more than once so defer plus an early stop cannot panic.
	stop := startBusySpinner(cachePath)
	require.NotNil(t, stop)
	stop()
	stop()

	require.NoError(t, os.WriteFile(cachePath, []byte("x"), 0o600))

	stop = startBusySpinner(cachePath)
	stop()
}
