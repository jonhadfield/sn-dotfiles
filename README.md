# sn-dotfiles

[![Tests](https://github.com/jonhadfield/sn-dotfiles/actions/workflows/tests.yml/badge.svg)](https://github.com/jonhadfield/sn-dotfiles/actions/workflows/tests.yml)
[![Latest release](https://img.shields.io/github/v/release/jonhadfield/sn-dotfiles)](https://github.com/jonhadfield/sn-dotfiles/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> Sync your dotfiles with an end-to-end encrypted [Standard Notes](https://standardnotes.com/) account.

## About

sn-dotfiles is a command-line tool to sync [dotfiles](https://www.thegeekyway.com/what-are-dotfiles/) with a [Standard Notes](https://standardnotes.com/) account.
It works by creating a root tag (default `dotfiles`, configurable via `root_tag`) and then maps dotfile directories with tags and dotfiles as notes.

## Why?

I wanted a simple way of securely storing, managing, and syncing my dotfiles across multiple machines. Standard Notes uses client-side encryption, so the contents of your dotfiles are encrypted before they leave your machine.

## Installation

On macOS and linux, using [homebrew](https://brew.sh):

```bash
brew install jonhadfield/tap/sn-dotfiles
```

That installs the `sn-dotfiles` binary and clears the macOS quarantine flag for
you.

**Or in one line**, which picks the right archive for your platform, verifies
its checksum against the published list, and installs to `/usr/local/bin`,
using sudo only if the install directory is not writable:

<!-- The point of these is that they are a single pasteable line, so they
     cannot be wrapped to the usual width. -->
<!-- markdownlint-disable MD013 -->

```bash
curl -fsSL https://raw.githubusercontent.com/jonhadfield/sn-dotfiles/main/install.sh | sh
```

Set `BIN_DIR` to install somewhere else, or `VERSION` to pin a release:

```bash
curl -fsSL https://raw.githubusercontent.com/jonhadfield/sn-dotfiles/main/install.sh | BIN_DIR="$HOME/.local/bin" sh
curl -fsSL https://raw.githubusercontent.com/jonhadfield/sn-dotfiles/main/install.sh | VERSION=0.2.0 sh
```

<!-- markdownlint-enable MD013 -->

If you would rather read it before running it, [install.sh](install.sh) is in
this repository.

**Or download the latest release** from the
[releases page](https://github.com/jonhadfield/sn-dotfiles/releases) and install
the binary from the archive matching your platform:

```bash
tar -xzf sn-dotfiles_Darwin_universal.tar.gz sn-dotfiles
sudo install -m 755 ./sn-dotfiles /usr/local/bin/sn-dotfiles && rm ./sn-dotfiles
```

The darwin binaries are not signed. A browser download is quarantined, so
Gatekeeper refuses to run it; homebrew and the install script both clear that
flag, but for a browser download clear it yourself:

```bash
xattr -d com.apple.quarantine /usr/local/bin/sn-dotfiles
```

## Quick start

A config file is required, so create one before the first run. This example
tracks your git config and your neovim configuration:

```bash
mkdir -p ~/.config/sn-dotfiles
cat > ~/.config/sn-dotfiles/config.yaml <<'EOF'
include:
  - '^\.gitconfig$'
  - '^\.config/nvim/'
EOF
```

Then authenticate, start tracking a file, and see where things stand:

```bash
sn-dotfiles session --add     # stores a session in your keychain
sn-dotfiles add ~/.gitconfig  # copies it into Standard Notes as a note
sn-dotfiles status            # compares local files with the remote notes
sn-dotfiles sync --dry-run    # shows what a sync would do
sn-dotfiles sync              # does it
```

## Authentication

By default, your credentials will be requested every time, but you can store them using either environment variables or, on MacOS and Linux, store your session using the native Keychain application.

### Session (macOS Keychain / Gnome Keyring)

Using a session is different from storing credentials as you no longer need to authenticate. As a result, if using 2FA (Two Factor Authentication), you won't need to enter your token value each time.

```bash
sn-dotfiles session --add   # session will be stored after successful authentication
```

To encrypt your session when adding, pass a key, or `.` to hide its input:

```bash
sn-dotfiles session --add --session-key <key>
```

Prefix any command with `--use-session` to automatically retrieve and use the session.
If your session is encrypted, you will be prompted for the session key. To specify the key on the command line:

```bash
sn-dotfiles --use-session --session-key <key> <command>
```

### Environment variables

Note: if using 2FA, the token value will be requested each time. Your password
is your account's encryption password, so putting it in a shell profile leaves
it in plain text on disk; prefer a session for interactive use.

```bash
export SN_EMAIL=<email address>
export SN_PASSWORD=<password>
export SN_SERVER=<https://myserver.example.com>   # optional, if running personal server
export SN_USE_SESSION=true                        # same as passing --use-session
export SN_DEBUG=true                              # same as passing --debug
```

## Configuration

A config file is required. By default it is read from `$XDG_CONFIG_HOME/sn-dotfiles/config.yaml`, falling back to `~/.config/sn-dotfiles/config.yaml`; use `--config <path>` to read another file.

It lists regular expressions that decide which dotfiles are synced, and optionally a root tag that names the set in Standard Notes:

```yaml
root_tag: PersonalDotfiles  # optional; default is "dotfiles". Must not contain '.'
include:                    # required: sync paths matching at least one of these
  - '^\.gitconfig$'
  - '^\.config/fish/'
exclude:                    # optional: never sync paths matching any of these
  - '\.swp$'
```

**Root tag:** each machine syncs against one Standard Notes root tag (for example `PersonalDotfiles` or `WorkDotfiles`). Nested directory tags are built under that root (`PersonalDotfiles.config.fish`). On a new device, set `root_tag` (or pass `--root-tag`) and run `sync` to pull the latest files under that root. Use `sn-dotfiles root-tags` to list candidate roots already in the account. Multiple roots can coexist in one account; only the active root is synced.

To override the root tag for a single run:

```bash
sn-dotfiles --root-tag WorkDotfiles sync
```

Patterns are matched against each file's path relative to your home directory, using `/` as the separator, for example `.config/fish/config.fish`. To match everything in a folder, match its path as a prefix, e.g. `^\.config/fish/`.

- `status`, `sync` and `diff` only show and sync matching files. Notes in Standard Notes that don't match are left untouched.
- `add` skips files that don't match, so it never tracks something that would not be synced.
- `remove` and `wipe` are not filtered, so anything can still be removed (they still respect the active root tag).

To replace either list for a single run, pass the patterns before the command:

```bash
sn-dotfiles --include-regex '^\.config/nvim/' --exclude-regex '\.bak$' status
```

## Commands

| Command | Description |
|---|---|
| `status` | Compare tracked dotfiles with the remote notes |
| `sync` | Push newer local files, pull newer remote ones |
| `add` | Start tracking file(s) |
| `remove` | Stop tracking file(s), leaving the local files alone |
| `diff` | Show the differences between local files and remote notes |
| `session` | Add or remove a stored session |
| `root-tags` | List candidate root tags already in the account |
| `wipe` | Remove every dotfiles note and tag from the account |

Global flags: `--config`, `--home-dir`, `--server`, `--use-session`, `--session-key`, `--include-regex`, `--exclude-regex`, `--debug`, `--quiet`, `--no-stdout`.

### status

```bash
sn-dotfiles status
```

Status compares each tracked dotfile with its remote note and reports one of five states, which tell you what a sync would then do:

| State | Meaning | What `sync` does |
|---|---|---|
| `identical` | Local file and note match | Nothing |
| `local newer` | The local file changed most recently | Pushes the local file |
| `remote newer` | The note changed most recently | Overwrites the local file |
| `local missing` | Tracked remotely, absent locally | Creates the local file |
| `untracked` | Present locally, not in Standard Notes | Nothing, until you `add` it |

### add

```bash
sn-dotfiles add ~/.file1 ~/.dir1/file2
```

Add will take a copy of the specified file(s) and convert the files to Notes and each path to a Tag. Files that don't match the [configuration](#configuration) patterns are skipped. The above command would generate the following structure (with the default root tag):

```
dotfiles           <- root tag
    - .file1       <- note 
    - dir1         <- tag
        - file2    <- note
```

`add --all` tracks the dotfiles in the top level of your home directory. It is not recursive, so `~/.config/...` is not included.

### sync

```bash
sn-dotfiles sync --exclude ~/.file1
```

Sync will compare any dotfiles currently tracked under the active root tag in Standard Notes with their local equivalents and:
- Update the filesystem dotfile if the remote was updated more recently
- Update the remote if the filesystem dotfile is newer
- Create any missing dotfiles and paths that exist remotely  

The example command would sync the ~/.dir1 path and the file it contains, but ignore ~/.file1.
Only files matching the [configuration](#configuration) patterns are synced. 

To see what a sync would do before letting it do anything, add `--dry-run`:

```bash
sn-dotfiles sync --dry-run
```

```
.zshrc      | would push
.vimrc      | would pull
.gitconfig  | unchanged

dry run: nothing was written (1 to push, 1 to pull)
```

It compares exactly as a real sync does, honouring `--exclude`, any paths you
name and the configuration patterns, then stops before writing anything.

### remove

```bash
sn-dotfiles remove ~/.dir1
```

Remove will recursively (if path specified) remove the remote Notes for the specified filesystem path.
In the above example, the Note file2 and the Tag dir1 will be deleted. Remove will never change files on the filesystem.

### root-tags

```bash
sn-dotfiles root-tags
```

Lists candidate root tags already present in the Standard Notes account (undotted tags that look like dotfile roots). The active root from config or `--root-tag` is marked `(active)`.

### diff

```bash
sn-dotfiles diff ~/.dir1
```

Diff writes the local file and the remote note to temporary files and runs your system's `diff` command on them, so `diff` must be on your `PATH`. The temporary files are removed afterwards.

### wipe

```bash
sn-dotfiles wipe
```

Wipe deletes every dotfiles note and tag in the account. It shows the account email and asks for confirmation first; `--force` skips the prompt. It does not touch your local files.

## What is not synced

- **Only files under your home directory** whose path relative to it starts with a dot. Anything else is rejected with `is not a valid dotfile path`.
- **Symlinks are an error, not a skip**: `symlink not supported`. The same goes for sockets, devices, named pipes and other irregular files.
- **Files over 10MB**, rejected with `file too large`.
- **Binary files**, reported as `skipped: binary file`. A note stores text, so a binary file would come back different to how it went in. `add` leaves it untracked, and `sync` skips a tracked file that has since become binary, rather than overwriting the note with content it cannot store. A file counts as binary if its first 8000 bytes contain a NUL byte or are not valid UTF-8.
- **Anything not matching your include patterns**, or matching an exclude pattern.

Two things worth knowing about what a pull does:

- **File modes are not preserved.** A file created by a pull gets your default permissions (usually 0644), not the mode it had on the machine it came from. Check anything sensitive, such as `~/.netrc` or an ssh config, after pulling it onto a new machine.
- **Notes are text.** Your dotfiles are stored as note content, so this is a tool for text configuration, not for keys, images or databases.

## Troubleshooting

- **`config file ... not found`** — every command except `session` needs a config file. See [Quick start](#quick-start).
- **`symlink not supported`** — sn-dotfiles will not follow a symlinked dotfile. Track the file it points at instead.
- **`the following notes and tags are overlapping`** — a note and a tag in your account describe the same path, for example a note `.config` and a tag `.config`. Rename or remove one of them in the Standard Notes app.
- **Stale or confusing local state** — the cache lives at `~/.sn-dotfiles/sn-dotfiles-<hash>.db`, one file per account. Deleting it forces a fresh download on the next run.
- **Anything unexplained** — `--debug` prints the API calls and the comparison decisions.

## Bash autocompletion

The bash completion tool is installed by default on most Linux distributions. On macOS, install it with homebrew:

```bash
brew install bash-completion
```

Then add the following to `~/.bash_profile`:

```bash
[ -f "$(brew --prefix)/etc/profile.d/bash_completion.sh" ] && . "$(brew --prefix)/etc/profile.d/bash_completion.sh"
```

Install the [completion script](autocomplete/bash_autocomplete), naming it after the command so bash picks it up:

```bash
# macOS
cp autocomplete/bash_autocomplete "$(brew --prefix)/etc/bash_completion.d/sn-dotfiles"

# Linux
sudo cp autocomplete/bash_autocomplete /etc/bash_completion.d/sn-dotfiles
```

Then `sn-dotfiles <tab>` completes commands and flags.

## Known issues

- Notes moved to trash using the Standard Notes app will still be managed by sn-dotfiles until they are permanently deleted
