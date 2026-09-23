package sndotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gosn "github.com/jonhadfield/gosn-v2/items"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestDefaultConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

	path, err := DefaultConfigPath()
	require.NoError(t, err)
	require.Equal(t, "/tmp/xdg/sn-dotfiles/config.yaml", path)

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/home")

	path, err = DefaultConfigPath()
	require.NoError(t, err)
	require.Equal(t, "/tmp/home/.config/sn-dotfiles/config.yaml", path)
}

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    Config
		wantErr string
	}{
		{
			name:    "include and exclude",
			content: "include:\n  - '^\\.gitconfig$'\n  - '^\\.config/fish/'\nexclude:\n  - '\\.swp$'\n",
			want:    Config{Include: []string{`^\.gitconfig$`, `^\.config/fish/`}, Exclude: []string{`\.swp$`}},
		},
		{
			name:    "with root_tag",
			content: "root_tag: PersonalDotfiles\ninclude:\n  - '.*'\n",
			want:    Config{RootTag: "PersonalDotfiles", Include: []string{".*"}},
		},
		{
			name:    "invalid root_tag with dot",
			content: "root_tag: Personal.Dotfiles\ninclude:\n  - '.*'\n",
			wantErr: "must not contain '.'",
		},
		{
			name:    "include only",
			content: "include:\n  - '.*'\n",
			want:    Config{Include: []string{".*"}},
		},
		{
			name:    "unknown key",
			content: "includes:\n  - '.*'\n",
			wantErr: "invalid config file",
		},
		{
			name:    "invalid yaml",
			content: "include: [\n",
			wantErr: "failed to read config file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadConfig(writeConfig(t, tc.content))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, cfg)
		})
	}
}

func TestLoadConfigMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := LoadConfig(path)
	require.ErrorContains(t, err, fmt.Sprintf("config file %s not found", path))
	require.ErrorContains(t, err, "include:")
}

func TestNewPathFilter(t *testing.T) {
	_, err := NewPathFilter(nil, []string{"x"})
	require.ErrorContains(t, err, "at least one include pattern is required")

	_, err = NewPathFilter([]string{"("}, nil)
	require.ErrorContains(t, err, `invalid include pattern "("`)

	_, err = NewPathFilter([]string{".*"}, []string{"["})
	require.ErrorContains(t, err, `invalid exclude pattern "["`)
}

func TestPathFilterMatch(t *testing.T) {
	filter, err := NewPathFilter([]string{`^\.gitconfig$`, `^\.config/fish/`}, []string{`\.swp$`})
	require.NoError(t, err)

	testCases := []struct {
		path string
		want bool
	}{
		{path: ".gitconfig", want: true},
		{path: ".gitconfig.bak", want: false},
		{path: ".config/fish/config.fish", want: true},
		{path: ".config/fish/functions/ls.fish", want: true},
		{path: ".config/fish/.config.fish.swp", want: false},
		{path: ".config/nvim/init.lua", want: false},
		{path: ".ssh/config", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, tc.want, filter.Match(tc.path))
		})
	}

	var nilFilter *PathFilter
	require.True(t, nilFilter.Match(".anything"))
}

func TestCompareWithFilter(t *testing.T) {
	home := getTemporaryHome()
	twn, fwc := testCompareSetup1and2(home)
	fwc[fmt.Sprintf("%s/.untracked-fruit/cherry", home)] = "cherry content"
	require.NoError(t, createTemporaryFiles(fwc))

	defer func() {
		_ = os.RemoveAll(home)
	}()

	filter, err := NewPathFilter([]string{`^\.sn-dotfiles-test-fruit/`, `^\.untracked-fruit/`}, []string{`/lemon$`})
	require.NoError(t, err)

	paths := []string{
		fmt.Sprintf("%s/.sn-dotfiles-test-fruit", home),
		fmt.Sprintf("%s/.untracked-fruit", home),
	}

	diffs, err := compare(twn, home, paths, []string{}, filter, "", true)
	require.NoError(t, err)

	var got []string
	for _, d := range diffs {
		got = append(got, d.homeRelPath)
	}

	require.ElementsMatch(t, []string{
		".sn-dotfiles-test-fruit/apple",
		".sn-dotfiles-test-fruit/grape",
		".untracked-fruit/cherry",
	}, got)
}

func TestGenerateTagItemMapSkipsFilteredPaths(t *testing.T) {
	home := getTemporaryHome()
	gitConfigPath := fmt.Sprintf("%s/.gitconfig", home)
	sshConfigPath := fmt.Sprintf("%s/.ssh/config", home)
	require.NoError(t, createTemporaryFiles(map[string]string{
		gitConfigPath: "git config content",
		sshConfigPath: "ssh config content",
	}))

	defer func() {
		_ = os.RemoveAll(home)
	}()

	filter, err := NewPathFilter([]string{`^\.gitconfig$`}, nil)
	require.NoError(t, err)

	statusLines, tim, added, existing, skipped, err := generateTagItemMap([]string{gitConfigPath, sshConfigPath}, home, tagsWithNotes{}, filter, "")
	require.NoError(t, err)
	require.Equal(t, []string{gitConfigPath}, added)
	require.Empty(t, existing)
	require.Equal(t, []string{sshConfigPath}, skipped)
	require.Len(t, tim[DotFilesTag], 1)
	require.Equal(t, ".gitconfig", tim[DotFilesTag][0].(*gosn.Note).Content.GetTitle())
	require.NotContains(t, tim, DotFilesTag+".ssh")
	require.Len(t, statusLines, 2)
}

func TestLoadConfigMissingFileIsIdentifiable(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-config.yaml")

	_, err := LoadConfig(missing)
	require.Error(t, err)
	// commands that can run without configuration rely on this
	require.ErrorIs(t, err, ErrConfigNotFound)
	require.Contains(t, err.Error(), missing)
}

func TestLoadConfigBrokenFileIsNotMistakenForMissing(t *testing.T) {
	dir := t.TempDir()

	invalidYAML := filepath.Join(dir, "invalid.yaml")
	require.NoError(t, os.WriteFile(invalidYAML, []byte("include: [unclosed\n"), 0o600))

	_, err := LoadConfig(invalidYAML)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrConfigNotFound)

	badRoot := filepath.Join(dir, "bad-root.yaml")
	require.NoError(t, os.WriteFile(badRoot, []byte("root_tag: has.dot\ninclude:\n  - '^\\.gitconfig$'\n"), 0o600))

	_, err = LoadConfig(badRoot)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrConfigNotFound)
	require.Contains(t, err.Error(), "must not contain '.'")
}
