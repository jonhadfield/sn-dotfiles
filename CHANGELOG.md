# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-09-26

### Added

- `status` and `sync` warn when a Standard Notes editor is associated with a
  tracked dotfile. Notes hold plain text, and an editor that stores anything
  else — Super keeps JSON, Rich Text keeps HTML — rewrites the note when it
  saves, so the file pulled back would not be the file pushed. Nothing is
  changed remotely; the warning names the file and the editor

### Fixed

- A relative path was validated against the working directory rather than the
  home directory it had been resolved against, so `add .gitconfig` failed with
  `lstat .gitconfig: no such file or directory` for a file that existed
- Removing a path that was never tracked deleted the root tag and untracked
  everything beneath it. The root tag is now removed only when it holds no
  notes of its own and every child tag is going too
- `remove` reported an error when it found nothing to remove, rather than
  saying the path was not tracked
- A failed read while comparing a note with its file exited the process
  through `log.Fatal` instead of reporting the problem
- `wipe` accepted an invalid session, failing later and less clearly

### Changed

- The test suite runs without a Standard Notes account, against an in-memory
  server that exercises the real sign-in, encryption and sync paths: 115 tests,
  none skipped. CI runs against it, and a scheduled job runs the same suite
  against a real account daily
- Dependabot now raises version updates, not only security ones
- Conflicting note and tag paths are reported in a stable order
- Dropped the `fatih/set` dependency

## [0.3.0] - 2026-09-24

### Added

- Root tags: each machine syncs against a named root instead of the hardcoded
  `dotfiles` tag, set with `root_tag` in the config file or `--root-tag` on the
  command line. Several sets can live in one account and only the active one is
  synced
- `root-tags`, which lists the roots already present in an account, for pointing
  a new machine at an existing set
- `sync --dry-run`, which reports what a sync would push, pull and leave alone,
  then stops before writing anything. It compares exactly as a real sync does,
  honouring `--exclude`, named paths and the configuration patterns

### Fixed

- Binary files are no longer tracked or pushed. Note content is text, so a
  binary file came back corrupted: the push appeared to succeed, and pulling it
  wrote something different to what went in. `add` now leaves such a file
  untracked and `sync` skips a tracked file that has since become binary, both
  saying so rather than passing over it silently
- `--cachedb-dir` was never registered as a flag, so passing it failed with
  "flag provided but not defined"; the value read from configuration was also
  overwritten by the empty flag
- `--home-dir` was ignored by `add`, which resolved the real home directory
  instead and rejected paths under the home just named
- Help output named the program `-` rather than `sn-dotfiles`, in both the main
  and per-command help
- The `sync --exclude` usage text read "exlude"

### Changed

- `wipe` names the root tag it is about to remove in its confirmation prompt
- The README documents the `status` states, what is not synced, and a quickstart
  that creates the required config file before the first command needs it
- Live-server tests are opt-in through `SN_INTEGRATION_TESTS`, so `go test ./...`
  runs the unit tests on a machine with no credentials. CI sets the variable and
  still runs the full suite
- CI queues test runs rather than cancelling them, so a merge no longer aborts
  an open pull request's run and leaves its test data behind for the next one

## [0.2.0] - 2026-09-20

### Added

- Homebrew installation: `brew install jonhadfield/tap/sn-dotfiles`, which also
  clears the macOS quarantine flag
- `install.sh`, a one-line installer that picks the archive for the platform,
  verifies its checksum against the published list and installs to
  `/usr/local/bin`, with `BIN_DIR` and `VERSION` overrides
- A tag-triggered release workflow, replacing manual releases

### Changed

- gosn-v2 updated, bringing fewer redundant sync requests and
  `golang.org/x/crypto` v0.56.0
- Go 1.26, in both the module and CI
- `google.golang.org/protobuf` raised to v1.36.12, fixing an infinite loop in
  `protojson.Unmarshal` on certain invalid input

### Removed

- `SNServerURL`, an exported constant naming a server that no longer exists. It
  had no remaining users; the server comes from `SN_SERVER` or gosn-v2's default

## Earlier releases

Releases up to and including 0.1.6 (2021) predate this file. See the
[releases page](https://github.com/jonhadfield/sn-dotfiles/releases) for their
notes.
