package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	dotfilesSN "github.com/jonhadfield/dotfiles-sn"
	keyring "github.com/zalando/go-keyring"

	"github.com/jonhadfield/gosn"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

// overwritten at build time
var version, versionOutput, tag, sha, buildDate string

const (
	msgSessionRemovalSuccess = "session removed successfully"
	msgSessionRemovalFailure = "failed to remove session"
)

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
		cli.BoolFlag{Name: "session-key"},
		cli.BoolFlag{Name: "quiet"},
	}
	app.CommandNotFound = func(c *cli.Context, command string) {
		_, _ = fmt.Fprintf(c.App.Writer, "\ninvalid command: \"%s\" \n\n", command)
		cli.ShowAppHelpAndExit(c, 1)
	}
	app.Commands = []cli.Command{
		{
			Name:  "status",
			Usage: "compare local and remote",

			Action: func(c *cli.Context) error {
				if !c.GlobalBool("quiet") {
					display = true
				}
				session, _, err := dotfilesSN.GetSession(c.GlobalBool("use-session"), c.GlobalString("server"))
				if err != nil {
					return err
				}

				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, msg, err = dotfilesSN.Status(session, home, c.Args(), c.GlobalBool("debug"))
				return err
			},
		},
		{
			Name:  "sync",
			Usage: "sync dotfiles",
			Flags: []cli.Flag{
				// TODO: not implemented
				cli.BoolFlag{
					Name:   "delete",
					Usage:  "remove remotes that don't exist locally",
					Hidden: true,
				},
				cli.StringSliceFlag{
					Name:  "exclude",
					Usage: "exlude path from sync",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.GlobalBool("quiet") {
					display = true
				}
				session, _, err := dotfilesSN.GetSession(c.GlobalBool("use-session"), c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, _, msg, err = dotfilesSN.Sync(session, home, c.Args(), c.StringSlice("exclude"), c.GlobalBool("debug"))
				if err != nil {
					return err
				}
				return err
			},
		},
		{
			Name:  "add",
			Usage: "start tracking file(s)",
			Action: func(c *cli.Context) error {
				if len(c.Args()) == 0 {
					_ = cli.ShowCommandHelp(c, "add")
					return nil
				}
				if !c.GlobalBool("quiet") {
					display = true
				}
				var invalidPaths bool
				for _, path := range c.Args() {
					if !isValidDotfilePath(path) {
						invalidPaths = true
						msg = fmt.Sprintf("\"%s\" is not a valid dotfile path\n", path)
						return nil
					}
				}
				if invalidPaths {
					return nil
				}
				session, _, err := dotfilesSN.GetSession(c.GlobalBool("use-session"), c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				ai := dotfilesSN.AddInput{Session: session, Home: home, Paths: c.Args(), Debug: c.GlobalBool("debug")}
				var ao dotfilesSN.AddOutput
				ao, err = dotfilesSN.Add(ai)

				if err != nil {
					return err
				}
				if ao.NotesPushed > 0 {
					msg = fmt.Sprintf("%d files added", ao.NotesPushed)
				} else {
					msg = "nothing to do"
				}
				return err

			},
		},
		{
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
				var invalidPaths bool
				for _, path := range c.Args() {
					if !isValidDotfilePath(path) {
						invalidPaths = true
						fmt.Printf("\"%s\" is not a valid dotfile path\n", path)
					}
				}
				if invalidPaths {
					return nil
				}
				session, _, err := dotfilesSN.GetSession(c.GlobalBool("use-session"), c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				var notesRemoved int
				notesRemoved, _, _, _, err = dotfilesSN.Remove(session, home, c.Args(), c.GlobalBool("debug"))
				if err != nil {
					return err
				}
				if notesRemoved > 0 {
					msg = fmt.Sprintf("%d files removed", notesRemoved)
				} else {
					msg = fmt.Sprintf("nothing to do")
				}
				return err
			},
		},
		{
			Name:  "diff",
			Usage: "display differences between local and remote",
			Action: func(c *cli.Context) error {
				if !c.GlobalBool("quiet") {
					display = true
				}
				session, _, err := dotfilesSN.GetSession(c.GlobalBool("use-session"),
					c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, msg, err = dotfilesSN.Diff(session, home, c.Args(), c.GlobalBool("debug"))
				return err
			},
		},
		{
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
					Name:  "session-key",
					Usage: "key to encrypt/decrypt session",
				},
			},
			Hidden: false,
			Action: func(c *cli.Context) error {
				if !c.GlobalBool("quiet") {
					display = true
				}
				sAdd := c.Bool("add")
				sRemove := c.Bool("remove")
				sStatus := c.Bool("status")
				sessKey := c.String("session-key")
				if sStatus || sRemove {
					if err = sessionExists(); err != nil {
						return err
					}
				}
				nTrue := numTrue(sAdd, sRemove, sStatus)
				if nTrue == 0 || nTrue > 1 {
					_ = cli.ShowCommandHelp(c, "session")
					os.Exit(1)
				}
				if sAdd {
					msg, err = addSession(c.GlobalString("server"), sessKey)
					return err
				}
				if sRemove {
					msg = removeSession()
					return nil
				}
				if sStatus {
					var s string
					s, err = dotfilesSN.GetSessionFromKeyring(sessKey)
					if err != nil {
						return err
					}
					var email string
					email, _, err = dotfilesSN.ParseSessionString(s)
					if err != nil {
						msg = fmt.Sprint("failed to parse session: ", err)
						return nil
					}
					msg = fmt.Sprint("session found: ", email)
				}
				return err
			},
		},
		{
			Name:  "wipe",
			Usage: "manage session credentials",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force",
					Usage: "assume user confirmation",
				},
			},
			Hidden: true,
			Action: func(c *cli.Context) error {
				if !c.GlobalBool("quiet") {
					display = true
				}
				session, email, err := dotfilesSN.GetSession(c.GlobalBool("use-session"),
					c.GlobalString("server"))
				if err != nil {
					return err
				}
				var proceed bool
				if c.Bool("force") {
					proceed = true
				} else {
					fmt.Printf("wipe all dotfiles for account %s? ", email)
					var input string
					_, err = fmt.Scanln(&input)
					if err == nil && dotfilesSN.StringInSlice(input, []string{"y", "yes"}, false) {
						proceed = true
					}
				}
				if proceed {
					var num int
					num, err = dotfilesSN.WipeDotfileTagsAndNotes(session, c.GlobalBool("quiet"))
					if err != nil {
						return err
					}
					msg = fmt.Sprintf("%d removed", num)
				} else {
					return nil
				}
				return err
			},
		},
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

func sessionExists() error {
	eS, err := keyring.Get(dotfilesSN.Service, dotfilesSN.KeyringApplicationName)
	if err != nil {
		return err
	}
	if len(eS) == 0 {
		return errors.New("session is empty")
	}
	return nil
}

func addSession(snServer, inKey string) (res string, err error) {
	// check if session exists in keyring
	var s string
	s, err = keyring.Get(dotfilesSN.Service, dotfilesSN.KeyringApplicationName)
	// only return an error if there's an issue accessing the keyring
	if err != nil && !strings.Contains(err.Error(), "secret not found in keyring") {
		return
	}
	if s != "" {
		fmt.Print("replace existing session (y|n): ")
		var resp string
		_, err := fmt.Scanln(&resp)
		if err != nil || strings.ToLower(resp) != "y" {
			return "", err
		}
	}
	var session gosn.Session
	var email string
	session, email, err = dotfilesSN.GetSessionFromUser(snServer)
	if err != nil {
		return fmt.Sprint("failed to get session: ", err), err
	}

	rS := makeSessionString(email, session)
	if inKey != "" {
		key := []byte(inKey)
		rS = dotfilesSN.Encrypt(key, makeSessionString(email, session))
	}
	err = keyring.Set(dotfilesSN.Service, dotfilesSN.KeyringApplicationName, rS)
	if err != nil {
		return fmt.Sprint("failed to set session: ", err), err
	}
	return "session added successfully", err
}

func removeSession() string {
	err := keyring.Delete(dotfilesSN.Service, dotfilesSN.KeyringApplicationName)
	if err != nil {
		return fmt.Sprintf("%s: %s", msgSessionRemovalFailure, err.Error())
	}
	return msgSessionRemovalSuccess
}

func makeSessionString(email string, session gosn.Session) string {
	return fmt.Sprintf("%s;%s;%s;%s;%s", email, session.Server, session.Token, session.Ak, session.Mk)
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
