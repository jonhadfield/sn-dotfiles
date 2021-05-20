package main

import (
	"errors"
	"fmt"
	"github.com/jonhadfield/gosn-v2"
	"github.com/jonhadfield/gosn-v2/cache"
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
	app.Usage = "dotfiles sn"
	app.Description = ""

	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug"},
		cli.StringFlag{Name: "server"},
		cli.StringFlag{Name: "home-dir"},
		cli.BoolFlag{Name: "use-session"},
		cli.StringFlag{Name: "session-key"},
		cli.IntFlag{Name: "page-size", Hidden: true, Value: sndotfiles.DefaultPageSize},
		cli.BoolFlag{Name: "quiet"},
	}
	app.CommandNotFound = func(c *cli.Context, command string) {
		_, _ = fmt.Fprintf(c.App.Writer, "\ninvalid command: \"%s\" \n\n", command)
		cli.ShowAppHelpAndExit(c, 1)
	}
	statusCmd := cli.Command{
		Name:  "status",
		Usage: "compare local and remote",
		Action: func(c *cli.Context) error {
			if !c.GlobalBool("quiet") {
				display = true
			}

			var session cache.Session
			session, _, err = cache.GetSession(c.GlobalBool("use-session"),
				c.GlobalString("session-key"), c.GlobalString("server"),
				c.GlobalBool("debug"))
			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(session, "", sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			session.CacheDBPath = cacheDBPath

			home := c.GlobalString("home-dir")
			if home == "" {
				home = getHome()
			}
			_, msg, err = sndotfiles.Status(&session, home, c.Args(),
				c.GlobalInt("page-size"), c.GlobalBool("debug"))
			return err
		},
	}

	syncCmd := cli.Command{
		Name:  "sync",
		Usage: "sync dotfiles",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "exlude path from sync",
			},
		},
		BashComplete: func(c *cli.Context) {
			syncTasks := []string{"--exclude"}
			for _, t := range syncTasks {
				fmt.Println(t)
			}
		},
		Action: func(c *cli.Context) error {
			if !c.GlobalBool("quiet") {
				display = true
			}

			var session cache.Session
			session, _, err = cache.GetSession(c.GlobalBool("use-session"),
				c.GlobalString("session-key"), c.GlobalString("server"), c.GlobalBool("debug"))
			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(session, "", sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			session.CacheDBPath = cacheDBPath

			home := c.GlobalString("home-dir")
			if home == "" {
				home = getHome()
			}
			var so sndotfiles.SyncOutput
			so, err = sndotfiles.Sync(sndotfiles.SNDotfilesSyncInput{
				Session:  &session,
				Home:     home,
				Paths:    c.Args(),
				Exclude:  c.StringSlice("exclude"),
				PageSize: c.GlobalInt("page-size"),
			})
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
			if !c.GlobalBool("quiet") {
				display = true
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
				if !isValidDotfilePath(ap) {
					msg = fmt.Sprintf("\"%s\" is not a valid dotfile path", path)
					return nil
				}
				absPaths = append(absPaths, ap)
			}

			var session cache.Session
			session, _, err = cache.GetSession(c.GlobalBool("use-session"),
				c.GlobalString("session-key"), c.GlobalString("server"), c.GlobalBool("debug"))
			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(session, "", sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			session.CacheDBPath = cacheDBPath

			home := c.GlobalString("home-dir")
			if home == "" {
				home = getHome()
			}

			ai := sndotfiles.AddInput{Session: &session, Home: home, Paths: absPaths,
				PageSize: c.GlobalInt("page-size"), All: c.Bool("all")}

			var ao sndotfiles.AddOutput

			ao, err = sndotfiles.Add(ai)
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
			if !c.GlobalBool("quiet") {
				display = true
			}

			if len(c.Args()) == 0 {
				msg = "error: paths not specified"
				_ = cli.ShowCommandHelp(c, "add")
				return nil
			}

			var session cache.Session
			session, _, err = cache.GetSession(c.GlobalBool("use-session"),
				c.GlobalString("session-key"), c.GlobalString("server"),
				c.GlobalBool("debug"))
			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(session, "", sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			session.CacheDBPath = cacheDBPath

			home := c.GlobalString("home-dir")
			if home == "" {
				home = getHome()
			}
			ri := sndotfiles.RemoveInput{
				Session:  &session,
				Home:     home,
				Paths:    c.Args(),
				PageSize: c.GlobalInt("page-size"),
				Debug:    c.GlobalBool("debug"),
			}

			var ro sndotfiles.RemoveOutput

			ro, err = sndotfiles.Remove(ri)
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
			if !c.GlobalBool("quiet") {
				display = true
			}
			var session cache.Session
			session, _, err = cache.GetSession(c.GlobalBool("use-session"),
				c.GlobalString("session-key"), c.GlobalString("server"), c.GlobalBool("debug"))
			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(session, "", sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			session.CacheDBPath = cacheDBPath

			home := c.GlobalString("home-dir")
			if home == "" {
				home = getHome()
			}
			_, msg, err = sndotfiles.Diff(&session, home, c.Args(), c.GlobalInt("page-size"), true)
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
			if !c.GlobalBool("quiet") {
				display = true
			}
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
				msg, err = gosn.AddSession(c.GlobalString("server"), sessKey, nil)
				return err
			}
			if sRemove {
				msg = gosn.RemoveSession(nil)
				return nil
			}
			if sStatus {
				msg, err = gosn.SessionStatus(sessKey, nil)
			}
			return err
		},
	}

	wipeCmd := cli.Command{
		Name:  "wipe",
		Usage: "manage session credentials",
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
			if !c.GlobalBool("quiet") {
				display = true
			}

			var email string
			var session cache.Session
			session, email, err = cache.GetSession(c.GlobalBool("use-session"),
				c.GlobalString("session-key"), c.GlobalString("server"),
				c.GlobalBool("debug"))
			var cacheDBPath string
			cacheDBPath, err = cache.GenCacheDBPath(session, "", sndotfiles.SNAppName)
			if err != nil {
				return err
			}
			session.CacheDBPath = cacheDBPath

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
				num, err = sndotfiles.WipeDotfileTagsAndNotes(&session, c.GlobalInt("page-size"),
					c.GlobalBool("quiet"))
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

	return msg, display, app.Run(args)
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

func isValidDotfilePath(path string) bool {
	home := getHome()

	dir, filename := filepath.Split(path)

	homeRelPath, err := stripHome(dir+filename, home)
	if err != nil {
		return false
	}

	return strings.HasPrefix(homeRelPath, ".")
}
