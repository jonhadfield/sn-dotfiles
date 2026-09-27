package sndotfiles

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jonhadfield/findexec"
	gosn "github.com/jonhadfield/gosn-v2/items"
	"golang.org/x/term"
)

// SyncChoice is the action to take when local and remote content differ.
type SyncChoice int

const (
	// SyncChoiceLocal pushes the local file to Standard Notes.
	SyncChoiceLocal SyncChoice = iota
	// SyncChoiceRemote pulls the remote note onto the filesystem.
	SyncChoiceRemote
	// SyncChoiceSkip leaves both sides alone.
	SyncChoiceSkip
)

// ErrSyncAborted is returned when the user quits an interactive sync before
// any changes are written.
var ErrSyncAborted = errors.New("sync aborted")

// ConflictChooser decides what to do when local and remote content differ.
// defaultAction is SyncChoiceLocal when the local file is newer, otherwise
// SyncChoiceRemote. Returning an error aborts the sync without writing.
type ConflictChooser func(diff ItemDiff, defaultAction SyncChoice) (SyncChoice, error)

const maxInteractiveDiffLines = 50

func defaultConflictChooser(diff ItemDiff, defaultAction SyncChoice) (SyncChoice, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return SyncChoiceSkip, errors.New("--interactive requires a terminal (stdin is not a tty)")
	}

	return promptConflict(diff, defaultAction, os.Stdin, os.Stderr)
}

func promptConflict(diff ItemDiff, defaultAction SyncChoice, in io.Reader, out io.Writer) (SyncChoice, error) {
	path := addDot(diff.homeRelPath)
	fmt.Fprintf(out, "\n%s differs (%s)\n", bold(path), colourDiff(diff.diff))

	if err := writeContentDiff(out, diff.local, diff.remote.Content.GetText(), maxInteractiveDiffLines); err != nil {
		return SyncChoiceSkip, err
	}

	defaultKey := "l"
	defaultLabel := "local (push)"

	if defaultAction == SyncChoiceRemote {
		defaultKey = "r"
		defaultLabel = "remote (pull)"
	}

	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprintf(out, "Keep [L]ocal (push), [R]emote (pull), [S]kip, [D]iff, or [Q]uit? [%s / %s] ",
			defaultKey, defaultLabel)

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return SyncChoiceSkip, err
			}

			return SyncChoiceSkip, ErrSyncAborted
		}

		choice, showDiff, err := parseConflictChoice(scanner.Text(), defaultAction)
		if err != nil {
			if errors.Is(err, ErrSyncAborted) {
				return SyncChoiceSkip, ErrSyncAborted
			}

			fmt.Fprintf(out, "%s\n", err.Error())

			continue
		}

		if showDiff {
			if err = writeContentDiff(out, diff.local, diff.remote.Content.GetText(), 0); err != nil {
				return SyncChoiceSkip, err
			}

			continue
		}

		return choice, nil
	}
}

// parseConflictChoice maps user input to a sync choice. Empty input selects
// defaultAction. showDiff is true when the user asked to redisplay the diff.
func parseConflictChoice(input string, defaultAction SyncChoice) (choice SyncChoice, showDiff bool, err error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "":
		return defaultAction, false, nil
	case "l", "local", "push":
		return SyncChoiceLocal, false, nil
	case "r", "remote", "pull":
		return SyncChoiceRemote, false, nil
	case "s", "skip":
		return SyncChoiceSkip, false, nil
	case "d", "diff":
		return SyncChoiceSkip, true, nil
	case "q", "quit":
		return SyncChoiceSkip, false, ErrSyncAborted
	default:
		return SyncChoiceSkip, false, fmt.Errorf("invalid choice %q: enter l, r, s, d, or q", input)
	}
}

// writeContentDiff prints a unified (or plain) diff of local vs remote to out.
// If maxLines > 0, output is truncated after that many lines.
func writeContentDiff(out io.Writer, local, remote string, maxLines int) error {
	diffBinary := findexec.Find("diff", "")
	if diffBinary == "" {
		fmt.Fprintln(out, yellow("(diff binary not found; content differs)"))

		return nil
	}

	tempDir := os.TempDir()
	if !strings.HasSuffix(tempDir, string(os.PathSeparator)) {
		tempDir += string(os.PathSeparator)
	}

	uuid := gosn.GenUUID()
	f1path := fmt.Sprintf("%ssn-dotfiles-interactive-%s-local", tempDir, uuid)
	f2path := fmt.Sprintf("%ssn-dotfiles-interactive-%s-remote", tempDir, uuid)

	if err := os.WriteFile(f1path, []byte(local), 0o600); err != nil {
		return err
	}

	defer func() { _ = os.Remove(f1path) }()

	if err := os.WriteFile(f2path, []byte(remote), 0o600); err != nil {
		return err
	}

	defer func() { _ = os.Remove(f2path) }()

	// Prefer unified diff; fall back to plain diff if -u is unsupported.
	cmd := exec.Command(diffBinary, "-u", "--label", "local", "--label", "remote", f1path, f2path)
	outBytes, oErr := cmd.CombinedOutput()

	if exitCode := diffExitCode(oErr); exitCode == 2 {
		cmd = exec.Command(diffBinary, f1path, f2path)
		outBytes, oErr = cmd.CombinedOutput()

		if exitCode = diffExitCode(oErr); exitCode == 2 {
			return fmt.Errorf("failed to compare local and remote content")
		}
	}

	text := string(outBytes)
	if maxLines > 0 {
		text = truncateLines(text, maxLines)
	}

	if strings.TrimSpace(text) != "" {
		fmt.Fprintln(out, text)
	}

	return nil
}

func diffExitCode(err error) int {
	if err == nil {
		return 0
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}

	return 2
}

func truncateLines(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}

	return strings.Join(lines[:maxLines], "\n") + "\n" + yellow(fmt.Sprintf("... (%d more lines; enter d to show full diff)", len(lines)-maxLines))
}
