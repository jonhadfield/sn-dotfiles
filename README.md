# SN-Dotfiles

[![Build Status][travisci-image]][travisci-url] [![Coverage Status][coverage-image]][coverage-url] [![Go Report Card][go-report-card-image]][go-report-card-url]  

## about

sn-dotfiles is a command-line tool to sync dotfiles with a [Standard Notes](https://standardnotes.org/) account.  
It works by creating a tag called 'dotfiles' and then maps dotfile directories with tags and dotfiles as notes.

## installation
Download the latest release here: https://github.com/jonhadfield/sn-dotfiles/releases

#### macOS and Linux

Install:  
``
$ install <downloaded binary> /usr/local/bin/sn-dotfiles
``  

## running

### credentials

By default, your credentials will be requested every time, but you can store them using either environment variables or, on MacOS and Linux, store your session using the native Keychain application.

#### environment variables
Note: if using 2FA, the token value will be requested each time
```
export SN_EMAIL=<email address>
export SN_PASSWORD=<password>
export SN_SERVER=<https://myserver.example.com>   # optional, if running personal server
```

#### session - macOS Keychain
Using a session is different from storing credentials as you no longer need to authenticate. As a result, if using 2FA, you won't need to enter your token value each time.  
##### add session
```
sn-dotfiles session --add   # session will be stored after successful authentication
```
##### using a session
Prefix any command with ```--use-session``` to automatically retrieve and use the session.

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