package sndotfiles

import (
	"os"
	stdsync "sync"
	"time"

	"github.com/briandowns/spinner"
)

// startBusySpinner shows a syncing/initializing spinner on stderr and returns a
// stop function that always clears it. Progress goes to stderr so result text
// on stdout is never mixed into the spinner line.
func startBusySpinner(cacheDBPath string) func() {
	prefix := HiWhite("syncing ")
	if _, err := os.Stat(cacheDBPath); os.IsNotExist(err) {
		prefix = HiWhite("initializing ")
	}

	s := spinner.New(
		spinner.CharSets[SpinnerCharSet],
		SpinnerDelay*time.Millisecond,
		spinner.WithWriter(os.Stderr),
	)
	s.Prefix = prefix
	s.FinalMSG = ""
	s.Start()

	var once stdsync.Once

	return func() {
		once.Do(func() {
			s.Stop()
		})
	}
}
