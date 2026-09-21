package main

import (
	"errors"
	"fmt"
	"github.com/jonhadfield/gosn-v2/cache"
	"github.com/jonhadfield/gosn-v2/common"
	"github.com/jonhadfield/gosn-v2/session"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sndotfiles "github.com/jonhadfield/dotfiles-sn/sn-dotfiles"

	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

// overwritten at build time
var version, versionOutput, tag, sha, buildDate string

type configOptsOutput struct {
	useStdOut  bool
	display    bool
	useSession bool
	home       string
	sessKey    string
	server     string
	pageSize   int
	cacheDBDir string
	debug      bool
}

func getOpts(c *cli.Context) (out configOptsOutput, err error) {
	out.useStdOut = c.Bool("no-stdout")

	if !c.GlobalBool("no-stdout") {
		out.useStdOut = false
	}

	if c.GlobalBool("use-session") || viper.GetBool("use_session") {
		out.useSession = true
	}

	out.sessKey = c.GlobalString("session-key")

	out.server = c.GlobalString("server")
	if viper.GetString("server") != "" {
		out.server = viper.GetString("server")
	}

	// the flag wins, then SN_CACHEDB_DIR, then the default the cache picks
	out.cacheDBDir = c.GlobalString("cachedb-dir")
	if out.cacheDBDir == "" {
		out.cacheDBDir = viper.GetString("cachedb_dir")
	}

	out.display = true
	if c.GlobalBool("quiet") {
		out.display = false
	}

	out.home = c.GlobalString("home-dir")
	if out.home == "" {
		out.home = getHome()
	}

	out.pageSize = c.GlobalInt("page-size")

	out.debug = viper.GetBool("debug")
	if c.GlobalBool("debug") {
		out.debug = true
	}

	return
}

func main() {
	msg, display, err := startCLI(os.Args)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
		os.Exit(1)
	}

	if display && msg != "" {
		fmt.Println(msg)
	}

	os.Exit(0)
}

func startCLI(args []string) (msg string, display bool, err error) {
	const funcName = "startCLI"

	viper.SetEnvPrefix("sn")

	err = viper.BindEnv("email")
	if err != nil {
		return "", false, err
	}

	err = viper.BindEnv("password")
	if err != nil {
		return "", false, err
	}

	err = viper.BindEnv("server")
	if err != nil {
		return "", false, err
	}

	err = viper.BindEnv("debug")
	if err != nil {
		return "", false, err
	}

	err = viper.BindEnv("use_session")
	if err != nil {
		return "", false, err
	}

	err = viper.BindEnv("cachedb_dir")
	if err != nil {
		return "", false, err
	}

	if tag != "" && buildDate != "" {
		versionOutput = fmt.Sprintf("[%s-%s] %s UTC", tag, sha, buildDate)
	} else {
		versionOutput = version
	}

	app := cli.NewApp()
	app.EnableBashCompletion = true

	app.Name = "sn-dotfiles"
	app.Version = versionOutput
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		{
			Name:  "Jon Hadfield",
			Email: "jon@lessknown.co.uk",
		},
	}
	app.HelpName = "-"
	app.Usage = "sync dotfiles with Standard Notes"
	app.Description = ""

	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug"},
		cli.StringFlag{Name: "server"},
		cli.StringFlag{Name: "home-dir"},
		cli.BoolFlag{Name: "use-session"},
		cli.StringFlag{Name: "session-key"},
		cli.IntFlag{Name: "page-size", Hidden: true, Value: sndotfiles.DefaultPageSize},
		cli.BoolFlag{Name: "quiet"},
		cli.BoolFlag{Name: "no-stdout"},
		cli.StringFlag{Name: "config", Usage: "path to config file (default: ~/.config/sn-dotfiles/config.yaml)"},
		cli.StringFlag{Name: "cachedb-dir", Usage: "directory holding the local cache database (default: ~/.sn-dotfiles)"},
		cli.StringSliceFlag{Name: "include-regex", Usage: "only sync paths matching this pattern, replacing the config file's include list"},
		cli.StringSliceFlag{Name: "exclude-regex", Usage: "never sync paths matching this pattern, replacing the config file's exclude list"},
	}
	app.CommandNotFound = func(c *cli.Context, command string) {
		_, _ = fmt.Fprintf(c.App.Writer, "\ninvalid command: \"%s\" \n\n", command)
		cli.ShowAppHelpAndExit(c, 1)
	}
	statusCmd := cli.Command{
		Name:  "status",
		Usage: "compare local and remote",
		Action: func(c *cli.Context) error {
			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			var filter *sndotfiles.PathFilter
			filter, err = loadFilter(c)
			if err != nil {
				return err
			}

			var sess cache.Session
			sess, _, err = cache.GetSession(common.NewHTTPClient(), opts.useSession, opts.sessKey, opts.server, opts.debug)
			if err != nil {
				return err
			}

			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(sess, opts.cacheDBDir, sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			sess.CacheDBPath = cacheDBPath

			_, msg, err = sndotfiles.Status(&sess, opts.home, c.Args(), filter, opts.pageSize, opts.debug, false)
			return err
		},
	}

	syncCmd := cli.Command{
		Name:  "sync",
		Usage: "sync dotfiles",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "exclude path from sync",
			},
			cli.BoolFlag{
				Name:  "dry-run",
				Usage: "show what sync would do, without changing anything",
			},
		},
		BashComplete: func(c *cli.Context) {
			syncTasks := []string{"--exclude", "--dry-run"}
			for _, t := range syncTasks {
				fmt.Println(t)
			}
		},
		Action: func(c *cli.Context) error {
			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			var filter *sndotfiles.PathFilter
			filter, err = loadFilter(c)
			if err != nil {
				return err
			}

			var sess cache.Session
			sess, _, err = cache.GetSession(common.NewHTTPClient(), opts.useSession,
				opts.sessKey, opts.server, opts.debug)
			if err != nil {
				return err
			}

			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(sess, opts.cacheDBDir, sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			sess.CacheDBPath = cacheDBPath

			var so sndotfiles.SyncOutput
			so, err = sndotfiles.Sync(sndotfiles.SNDotfilesSyncInput{
				Session:  &sess,
				Home:     opts.home,
				Paths:    c.Args(),
				Exclude:  c.StringSlice("exclude"),
				Filter:   filter,
				PageSize: opts.pageSize,
				Debug:    opts.debug,
				DryRun:   c.Bool("dry-run"),
			}, c.GlobalBool("no-stdout"))

			if err != nil {
				return err
			}
			msg = so.Msg

			return err
		},
	}

	addCmd := cli.Command{
		Name:  "add",
		Usage: "start tracking file(s)",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "all",
				Usage: "add all dotfiles (non-recursive)",
			},
		},
		Action: func(c *cli.Context) error {
			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			var filter *sndotfiles.PathFilter
			filter, err = loadFilter(c)
			if err != nil {
				return err
			}

			if !c.Bool("all") && len(c.Args()) == 0 {
				msg = "error: either specify paths to add or --all to add everything"
				_ = cli.ShowCommandHelp(c, "add")
				return nil
			}

			if c.Bool("all") && len(c.Args()) > 0 {
				msg = "error: specifying --all and paths does not make sense"
				_ = cli.ShowCommandHelp(c, "add")
				return nil
			}

			var absPaths []string
			for _, path := range c.Args() {
				var ap string
				ap, err = filepath.Abs(path)
				if err != nil {
					return err
				}
				if !isValidDotfilePath(ap, opts.home) {
					msg = fmt.Sprintf("\"%s\" is not a valid dotfile path", path)
					return nil
				}
				absPaths = append(absPaths, ap)
			}

			var sess cache.Session
			sess, _, err = cache.GetSession(common.NewHTTPClient(), opts.useSession,
				opts.sessKey, opts.server, opts.debug)
			if err != nil {
				return err
			}

			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(sess, opts.cacheDBDir, sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			sess.CacheDBPath = cacheDBPath

			ai := sndotfiles.AddInput{Session: &sess, Home: opts.home, Paths: absPaths,
				PageSize: opts.pageSize, All: c.Bool("all"), Filter: filter}

			var ao sndotfiles.AddOutput

			ao, err = sndotfiles.Add(ai, true)
			if err != nil {
				return err
			}

			msg = ao.Msg

			return err
		},
	}

	removeCmd := cli.Command{
		Name:  "remove",
		Usage: "stop tracking file(s)",
		Action: func(c *cli.Context) error {
			if len(c.Args()) == 0 {
				_ = cli.ShowCommandHelp(c, "remove")
				return nil
			}

			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			// config is required even though these commands don't filter
			if _, err = loadFilter(c); err != nil {
				return err
			}

			if len(c.Args()) == 0 {
				msg = "error: paths not specified"
				_ = cli.ShowCommandHelp(c, "add")
				return nil
			}

			var sess cache.Session
			sess, _, err = cache.GetSession(common.NewHTTPClient(), opts.useSession,
				opts.sessKey, opts.server,
				opts.debug)
			if err != nil {
				return err
			}

			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(sess, opts.cacheDBDir, sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			sess.CacheDBPath = cacheDBPath

			ri := sndotfiles.RemoveInput{
				Session:  &sess,
				Home:     opts.home,
				Paths:    c.Args(),
				PageSize: opts.pageSize,
				Debug:    opts.debug,
			}

			var ro sndotfiles.RemoveOutput

			ro, err = sndotfiles.Remove(ri, c.Bool("no-stdout"))
			if err != nil {
				return err
			}
			msg = ro.Msg

			return err
		},
	}

	diffCmd := cli.Command{
		Name:  "diff",
		Usage: "display differences between local and remote",
		Action: func(c *cli.Context) error {
			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			var filter *sndotfiles.PathFilter
			filter, err = loadFilter(c)
			if err != nil {
				return err
			}

			var sess cache.Session
			sess, _, err = cache.GetSession(common.NewHTTPClient(), opts.useSession,
				opts.sessKey, opts.server, opts.debug)
			if err != nil {
				return err
			}

			var cacheDBPath string

			cacheDBPath, err = cache.GenCacheDBPath(sess, opts.cacheDBDir, sndotfiles.SNAppName)
			if err != nil {
				return err
			}

			sess.CacheDBPath = cacheDBPath

			_, msg, err = sndotfiles.Diff(&sess, opts.home, c.Args(), filter, opts.pageSize, true, c.Bool("no-stdout"))

			return err
		},
	}

	sessionCmd := cli.Command{
		Name:  "session",
		Usage: "manage session credentials",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "add",
				Usage: "add session to keychain",
			},
			cli.BoolFlag{
				Name:  "remove",
				Usage: "remove session from keychain",
			},
			cli.BoolFlag{
				Name:  "status",
				Usage: "get session details",
			},
			cli.StringFlag{
				Name:     "session-key",
				Usage:    "[optional] key to encrypt/decrypt session (enter '.' to hide key input)",
				Required: false,
			},
		},
		Hidden: false,
		BashComplete: func(c *cli.Context) {
			tasks := []string{"--add", "--remove", "--status", "--session-key"}
			if c.NArg() > 0 {
				return
			}
			for _, t := range tasks {
				fmt.Println(t)
			}
		},
		Action: func(c *cli.Context) error {
			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			sAdd := c.Bool("add")
			sRemove := c.Bool("remove")
			sStatus := c.Bool("status")
			sessKey := c.String("session-key")

			nTrue := numTrue(sAdd, sRemove, sStatus)
			if nTrue == 0 || nTrue > 1 {
				_ = cli.ShowCommandHelp(c, "session")
				os.Exit(1)
			}
			if sAdd {
				msg, err = session.AddSession(nil, opts.server, sessKey, nil, opts.debug)
				return err
			}
			if sRemove {
				msg = session.RemoveSession(nil)
				return nil
			}
			if sStatus {
				msg, err = session.SessionStatus(sessKey, nil)
			}
			return err
		},
	}

	wipeCmd := cli.Command{
		Name:  "wipe",
		Usage: "remove all dotfiles",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force",
				Usage: "assume user confirmation",
			},
		},
		BashComplete: func(c *cli.Context) {
			tasks := []string{"--force"}
			if c.NArg() > 0 {
				return
			}
			for _, t := range tasks {
				fmt.Println(t)
			}
		},
		Hidden: true,
		Action: func(c *cli.Context) error {
			var opts configOptsOutput
			opts, err = getOpts(c)
			if err != nil {
				return err
			}
			display = opts.display

			// config is required even though these commands don't filter
			if _, err = loadFilter(c); err != nil {
				return err
			}

			var email string
			var sess cache.Session
			sess, email, err = cache.GetSession(common.NewHTTPClient(), opts.useSession,
				opts.sessKey, opts.server,
				opts.debug)
			if err != nil {
				return err
			}

			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(sess, opts.cacheDBDir, sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			sess.CacheDBPath = cacheDBPath

			var proceed bool
			if c.Bool("force") {
				proceed = true
			} else {
				fmt.Printf("wipe all dotfiles for account %s? ", email)
				var input string
				_, err = fmt.Scanln(&input)
				if err == nil && sndotfiles.StringInSlice(input, []string{"y", "yes"}, false) {
					proceed = true
				}
			}
			if proceed {
				var num int
				num, err = sndotfiles.WipeDotfileTagsAndNotes(&sess, opts.pageSize, c.Bool("no-stdout"))
				if err != nil {
					return err
				}
				msg = fmt.Sprintf("%d removed", num)
			} else {
				return nil
			}

			return err
		},
	}

	app.Commands = []cli.Command{
		statusCmd,
		syncCmd,
		addCmd,
		removeCmd,
		diffCmd,
		sessionCmd,
		wipeCmd,
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	err = app.Run(args)
	if err != nil {
		err = fmt.Errorf("%v: %v", funcName, err)
	}

	return msg, display, err
}

// loadFilter reads the required config file and applies any --include-regex and --exclude-regex overrides
func loadFilter(c *cli.Context) (*sndotfiles.PathFilter, error) {
	path := c.GlobalString("config")
	if path == "" {
		var err error

		if path, err = sndotfiles.DefaultConfigPath(); err != nil {
			return nil, err
		}
	}

	cfg, err := sndotfiles.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	include, exclude := cfg.Include, cfg.Exclude

	if patterns := c.GlobalStringSlice("include-regex"); len(patterns) > 0 {
		include = patterns
	}

	if patterns := c.GlobalStringSlice("exclude-regex"); len(patterns) > 0 {
		exclude = patterns
	}

	filter, err := sndotfiles.NewPathFilter(include, exclude)
	if err != nil {
		return nil, fmt.Errorf("config file %s: %w", path, err)
	}

	return filter, nil
}

func numTrue(in ...bool) (total int) {
	for _, i := range in {
		if i {
			total++
		}
	}

	return
}

func stripHome(in, home string) (res string, err error) {
	if home == "" {
		err = errors.New("home required")
		return
	}

	if in == "" {
		err = errors.New("path required")
		return
	}

	if in == home {
		return
	}

	if strings.HasPrefix(in, home) {
		return in[len(home)+1:], nil
	}

	return
}

func getHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("failed to get home directory")
		panic(err)
	}

	return home
}

// isValidDotfilePath reports whether path is a dotfile under home. home comes
// from the caller rather than the environment so that --home-dir is honoured.
func isValidDotfilePath(path, home string) bool {
	dir, filename := filepath.Split(path)

	homeRelPath, err := stripHome(dir+filename, home)
	if err != nil {
		return false
	}

	return strings.HasPrefix(homeRelPath, ".")
}
