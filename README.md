# SN-Dotfiles

[![Build Status][travisci-image]][travisci-url] [![Coverage Status][coverage-image]][coverage-url] [![Go Report Card][go-report-card-image]][go-report-card-url]  
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=jonhadfield_sn-dotfiles&metric=alert_status)](https://sonarcloud.io/dashboard?id=jonhadfield_sn-dotfiles)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=jonhadfield_sn-dotfiles&metric=coverage)](https://sonarcloud.io/dashboard?id=jonhadfield_sn-dotfiles)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=jonhadfield_sn-dotfiles&metric=security_rating)](https://sonarcloud.io/dashboard?id=jonhadfield_sn-dotfiles)
## Important
This is an early release to add support for SN accounts using encryption version 004 so please ensure you have a backup in case of any issues caused by this app.
This is only compatible with accounts either created since November 2020, or older accounts that have backed up, then restored to force an upgrade to 004.

Please create an issue if you receive any errors or notice anything unexpected.

## about

sn-dotfiles is a command-line tool to sync [dotfiles](https://www.thegeekyway.com/what-are-dotfiles/) with a [Standard Notes](https://standardnotes.org/) account.  
It works by creating a tag called 'dotfiles' and then maps dotfile directories with tags and dotfiles as notes.

## why?

I wanted a simple way of securely storing, managing, and syncing my dotfiles across multiple machines. Standard Notes uses client-side encryption and provides numerous editors (some require [extended subscription](https://standardnotes.org/extensions)).  

## installation

### macOS

```
$ curl -L -O https://github.com/jonhadfield/sn-dotfiles/releases/latest/download/sn-dotfiles_darwin_amd64  
$ install ./sn-dotfiles_darwin_amd64 /usr/local/bin/sn-dotfiles && rm ./sn-dotfiles_darwin_amd64
```

### Linux

```
$ curl -L -O https://github.com/jonhadfield/sn-dotfiles/releases/latest/download/sn-dotfiles_linux_amd64  
$ sudo install ./sn-dotfiles_linux_amd64 /usr/local/bin/sn-dotfiles && rm ./sn-dotfiles_linux_amd64
``` 

## running

### authentication

By default, your credentials will be requested every time, but you can store them using either environment variables or, on MacOS and Linux, store your session using the native Keychain application.

#### environment variables
Note: if using 2FA, the token value will be requested each time
```
export SN_EMAIL=<email address>
export SN_PASSWORD=<password>
export SN_SERVER=<https://myserver.example.com>   # optional, if running personal server
```

#### session (macOS Keychain / Gnome Keyring)
Using a session is different from storing credentials as you no longer need to authenticate. As a result, if using 2FA (Two Factor Authentication), you won't need to enter your token value each time.  
##### add session
```
sn-dotfiles session --add   # session will be stored after successful authentication
```
To encrypt your session when adding:
```
sn-dotfiles session --add --session-key   # either enter key as part of command, or '.' to hide its input
```
##### using a session
Prefix any command with ```--use-session``` to automatically retrieve and use the session.
If your session is encrypted, you will be prompted for the session key. To specify the key on the command line:
```
sn-dotfiles --use-session --session-key <key> <command>
```

### configuration
A config file is required. By default it is read from `~/.config/sn-dotfiles/config.yaml` (or `$XDG_CONFIG_HOME/sn-dotfiles/config.yaml`); use `--config <path>` to read another file.

It lists regular expressions that decide which dotfiles are synced:
```yaml
include:              # required: sync paths matching at least one of these
  - '^\.gitconfig$'
  - '^\.config/fish/'
exclude:              # optional: never sync paths matching any of these
  - '\.swp$'
```
Patterns are matched against each file's path relative to your home directory, using `/` as the separator, for example `.config/fish/config.fish`. To match everything in a folder, match its path as a prefix, e.g. `^\.config/fish/`.

- `status`, `sync` and `diff` only show and sync matching files. Notes in Standard Notes that don't match are left untouched.
- `add` skips files that don't match, so it never tracks something that would not be synced.
- `remove` and `wipe` are not filtered, so anything can still be removed.

To replace either list for a single run, pass the patterns before the command:
```
sn-dotfiles --include-regex '^\.config/nvim/' --exclude-regex '\.bak$' status
```

## commands

### add
example:
```
sn-dotfiles add /home/me/.file1 /home/me/.dir1/file2
```
Add will take a copy of the specified file(s) and convert the files to Notes and each path to a Tag. Files that don't match the [configuration](#configuration) patterns are skipped. The above command would generate the following structure:
```
dotfiles           <- tag
    - .file1       <- note 
    - dir1         <- tag
        - file2    <- note
```

### sync
example:
```
sn-dotfiles sync --exclude /home/me/.file1
```
Sync will compare any dotfiles currently tracked in Standard Notes with their local equivalents and:
- Update the filesystem dotfile if the remote was updated more recently
- Update the remote if the filesystem dotfile is newer
- Create any missing dotfiles and paths that exist remotely  

The example command would sync the /home/me/dir1 path and the file it contains, but ignore /home/me/.file1.
Only files matching the [configuration](#configuration) patterns are synced. 

### remove
example:
```
sn-dotfiles remove /home/me/.dir1
```
Remove will recursively (if path specified) remove the remote Notes for the specified filesystem path.
In the above example, the Note file2 and the Tag dir1 will be deleted. Remove will never change files on the filesystem.

### diff
example:
```
sn-dotfiles diff /home/me/.dir1
```
Diff will compare the filesystem with the remote and then use the diff tool to generate a list of differences.

[travisci-image]: https://travis-ci.org/jonhadfield/sn-dotfiles.svg?branch=master
[travisci-url]: https://travis-ci.org/jonhadfield/sn-dotfiles
[go-report-card-url]: https://goreportcard.com/report/github.com/jonhadfield/sn-dotfiles
[go-report-card-image]: https://goreportcard.com/badge/github.com/jonhadfield/sn-dotfiles
[coverage-image]: https://coveralls.io/repos/github/jonhadfield/sn-dotfiles/badge.svg?branch=master
[coverage-url]: https://coveralls.io/github/jonhadfield/sn-dotfiles?branch=master

## bash autocompletion

#### tool
the bash completion tool should be installed by default on most Linux installations.  

To install on macOS (Homebrew)  
``
$ brew install bash_completion  
``  
then add the following to ~/.bash_profile:  
``  
[ -f /usr/local/etc/bash_completion ] && . /usr/local/etc/bash_completion
`` 
#### installing completion script ([found here](https://github.com/jonhadfield/sn-dotfiles/tree/master/autocomplete/bash_autocomplete))
##### macOS  
``  
$ cp bash_autocomplete /usr/local/etc/bash_completion.d/sn-dotfiles
``  
##### Linux  
``
$ cp bash_autocomplete /etc/bash_completion.d/sn-dotfiles
``

##### autocomplete commands
``
$ sn-dotfiles <tab>
``

## known issues

- Notes moved to trash using the Standard Notes app will still be managed by sn-dotfiles until they are permanently deleted 
