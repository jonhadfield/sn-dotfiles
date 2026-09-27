package sndotfiles

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConflictChoice(t *testing.T) {
	tests := []struct {
		input         string
		defaultAction SyncChoice
		want          SyncChoice
		showDiff      bool
		err           error
	}{
		{input: "", defaultAction: SyncChoiceLocal, want: SyncChoiceLocal},
		{input: "", defaultAction: SyncChoiceRemote, want: SyncChoiceRemote},
		{input: "l", defaultAction: SyncChoiceRemote, want: SyncChoiceLocal},
		{input: "L", defaultAction: SyncChoiceRemote, want: SyncChoiceLocal},
		{input: "local", defaultAction: SyncChoiceRemote, want: SyncChoiceLocal},
		{input: "push", defaultAction: SyncChoiceRemote, want: SyncChoiceLocal},
		{input: "r", defaultAction: SyncChoiceLocal, want: SyncChoiceRemote},
		{input: "remote", defaultAction: SyncChoiceLocal, want: SyncChoiceRemote},
		{input: "pull", defaultAction: SyncChoiceLocal, want: SyncChoiceRemote},
		{input: "s", defaultAction: SyncChoiceLocal, want: SyncChoiceSkip},
		{input: "skip", defaultAction: SyncChoiceLocal, want: SyncChoiceSkip},
		{input: "d", defaultAction: SyncChoiceLocal, want: SyncChoiceSkip, showDiff: true},
		{input: "diff", defaultAction: SyncChoiceLocal, want: SyncChoiceSkip, showDiff: true},
		{input: "q", defaultAction: SyncChoiceLocal, err: ErrSyncAborted},
		{input: "quit", defaultAction: SyncChoiceLocal, err: ErrSyncAborted},
		{input: "nope", defaultAction: SyncChoiceLocal, err: errors.New("invalid")},
	}

	for _, tt := range tests {
		t.Run(tt.input+"/"+choiceName(tt.defaultAction), func(t *testing.T) {
			got, showDiff, err := parseConflictChoice(tt.input, tt.defaultAction)
			if tt.err != nil {
				require.Error(t, err)
				if errors.Is(tt.err, ErrSyncAborted) {
					assert.ErrorIs(t, err, ErrSyncAborted)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.showDiff, showDiff)
		})
	}
}

func TestPromptConflictAcceptsDefaultAndOverride(t *testing.T) {
	diff := ItemDiff{
		homeRelPath: ".gitconfig",
		diff:        localNewer,
		local:       "local content\n",
		remote:      createNote("gitconfig", "remote content\n"),
	}

	in := strings.NewReader("\n")
	var out bytes.Buffer

	choice, err := promptConflict(diff, SyncChoiceLocal, in, &out)
	require.NoError(t, err)
	assert.Equal(t, SyncChoiceLocal, choice)
	assert.Contains(t, out.String(), ".gitconfig")

	in = strings.NewReader("r\n")
	out.Reset()

	choice, err = promptConflict(diff, SyncChoiceLocal, in, &out)
	require.NoError(t, err)
	assert.Equal(t, SyncChoiceRemote, choice)
}

func TestPromptConflictQuit(t *testing.T) {
	diff := ItemDiff{
		homeRelPath: ".vimrc",
		diff:        remoteNewer,
		local:       "a\n",
		remote:      createNote("vimrc", "b\n"),
	}

	choice, err := promptConflict(diff, SyncChoiceRemote, strings.NewReader("q\n"), &bytes.Buffer{})
	assert.ErrorIs(t, err, ErrSyncAborted)
	assert.Equal(t, SyncChoiceSkip, choice)
}

func TestPromptConflictRedisplayThenChoose(t *testing.T) {
	diff := ItemDiff{
		homeRelPath: ".zshrc",
		diff:        localNewer,
		local:       "one\n",
		remote:      createNote("zshrc", "two\n"),
	}

	in := strings.NewReader("d\ns\n")
	var out bytes.Buffer

	choice, err := promptConflict(diff, SyncChoiceLocal, in, &out)
	require.NoError(t, err)
	assert.Equal(t, SyncChoiceSkip, choice)
}

func TestSyncRejectsInteractiveWithDryRun(t *testing.T) {
	_, err := Sync(SNDotfilesSyncInput{
		DryRun:      true,
		Interactive: true,
	}, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--interactive cannot be used with --dry-run")
}

func choiceName(c SyncChoice) string {
	switch c {
	case SyncChoiceLocal:
		return "local"
	case SyncChoiceRemote:
		return "remote"
	default:
		return "skip"
	}
}
