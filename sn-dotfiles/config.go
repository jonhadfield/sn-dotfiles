package sndotfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/viper"
)

// Config defines the settings read from the sn-dotfiles config file
type Config struct {
	// RootTag is the Standard Notes root tag for this machine's dotfile set.
	// Empty defaults to DotFilesTag ("dotfiles"). Must not contain '.'.
	RootTag string `mapstructure:"root_tag"`
	// Include lists regular expressions of home-relative paths to sync; at least one is required
	Include []string `mapstructure:"include"`
	// Exclude lists regular expressions of home-relative paths never to sync
	Exclude []string `mapstructure:"exclude"`
}

const exampleConfig = `root_tag: dotfiles   # optional; default is "dotfiles". Examples: PersonalDotfiles, WorkDotfiles
include:
  - '^\.gitconfig$'
  - '^\.config/fish/'
exclude:
  - '\.swp$'`

// DefaultConfigPath returns the path of the config file: $XDG_CONFIG_HOME/sn-dotfiles/config.yaml,
// falling back to ~/.config/sn-dotfiles/config.yaml
func DefaultConfigPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		dir = filepath.Join(home, ".config")
	}

	return filepath.Join(dir, SNAppName, "config.yaml"), nil
}

// ErrConfigNotFound is returned by LoadConfig when the config file is absent.
// Commands that can run without one test for it with errors.Is, so that a
// config file which exists but is broken is still reported rather than ignored.
var ErrConfigNotFound = errors.New("config file not found")

// configNotFoundError carries the full message, which names the path and shows
// an example config, while unwrapping to ErrConfigNotFound for errors.Is.
type configNotFoundError struct {
	detail string
}

func (e *configNotFoundError) Error() string { return e.detail }

func (e *configNotFoundError) Unwrap() error { return ErrConfigNotFound }

// LoadConfig reads and validates the config file at path
func LoadConfig(path string) (Config, error) {
	var cfg Config

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, &configNotFoundError{
				detail: fmt.Sprintf("config file %s not found; create it with at least one include pattern, for example:\n\n%s", path, exampleConfig),
			}
		}

		return cfg, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := v.UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("invalid config file %s: %w", path, err)
	}

	if cfg.RootTag != "" {
		if _, err := ResolveRootTag(cfg.RootTag); err != nil {
			return cfg, fmt.Errorf("invalid config file %s: %w", path, err)
		}
	}

	return cfg, nil
}

// PathFilter decides which home-relative paths are synced
type PathFilter struct {
	include []*regexp.Regexp
	exclude []*regexp.Regexp
}

// NewPathFilter compiles the include and exclude patterns; at least one include pattern is required
func NewPathFilter(include, exclude []string) (*PathFilter, error) {
	if len(include) == 0 {
		return nil, errors.New("at least one include pattern is required")
	}

	var f PathFilter

	var err error

	if f.include, err = compilePatterns("include", include); err != nil {
		return nil, err
	}

	if f.exclude, err = compilePatterns("exclude", exclude); err != nil {
		return nil, err
	}

	return &f, nil
}

func compilePatterns(kind string, patterns []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, 0, len(patterns))

	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid %s pattern %q: %w", kind, p, err)
		}

		res = append(res, r)
	}

	return res, nil
}

// Match reports whether a home-relative path, such as .config/fish/config.fish, should be synced:
// it must match at least one include pattern and no exclude pattern.
// A nil filter matches every path.
func (f *PathFilter) Match(homeRelPath string) bool {
	if f == nil {
		return true
	}

	p := filepath.ToSlash(homeRelPath)

	for _, r := range f.exclude {
		if r.MatchString(p) {
			return false
		}
	}

	for _, r := range f.include {
		if r.MatchString(p) {
			return true
		}
	}

	return false
}

func (f *PathFilter) filterDiffs(diffs []ItemDiff) []ItemDiff {
	if f == nil {
		return diffs
	}

	var res []ItemDiff

	for _, d := range diffs {
		if f.Match(d.homeRelPath) {
			res = append(res, d)
		}
	}

	return res
}
