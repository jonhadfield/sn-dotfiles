package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	dotfilesSN "github.com/jonhadfield/dotfiles-sn"
	"github.com/jonhadfield/gosn"
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
	}
	app.CommandNotFound = func(c *cli.Context, command string) {
		_, _ = fmt.Fprintf(c.App.Writer, "\ninvalid command: \"%s\" \n\n", command)
		cli.ShowAppHelpAndExit(c, 1)
	}
	app.Commands = []cli.Command{
		{
			Name:  "status",
			Usage: "compare local and remote",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "quiet",
					Usage: "suppress all output",
				},
			},
			Action: func(c *cli.Context) error {
				var session gosn.Session
				session, _, err = getSession(c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, err = dotfilesSN.Status(session, home, c.Args(), c.Bool("quiet"), c.GlobalBool("debug"))
				return err
			},
		},
		{
			Name:  "sync",
			Usage: "sync dotfiles",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "delete",
					Usage: "remove remotes that don't exist locally",
				},
				cli.BoolFlag{
					Name:  "quiet",
					Usage: "suppress all output",
				},
			},
			Action: func(c *cli.Context) error {
				var session gosn.Session
				session, _, err = getSession(c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, _, err = dotfilesSN.Sync(session, home, c.Bool("quiet"), c.GlobalBool("debug"))
				if err != nil {
					return err
				}
				return err
			},
		},
		{
			Name:  "add",
			Usage: "start tracking file(s)",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "quiet",
					Usage: "suppress all output",
				},
			},
			Action: func(c *cli.Context) error {
				if len(c.Args()) == 0 {
					_ = cli.ShowCommandHelp(c, "add")
					return nil
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
				var session gosn.Session
				session, _, err = getSession(c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, _, _, err = dotfilesSN.Add(session, home, c.Args(), c.Bool("quiet"), c.GlobalBool("debug"))

				if err != nil {
					return err
				}
				return err
			},
		},
		{
			Name:  "remove",
			Usage: "stop tracking file(s)",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "quiet",
					Usage: "suppress all output",
				},
			},
			Action: func(c *cli.Context) error {
				if len(c.Args()) == 0 {
					_ = cli.ShowCommandHelp(c, "remove")
					return nil
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
				var session gosn.Session
				session, _, err = getSession(c.GlobalString("server"))
				if err != nil {
					return err
				}
				home := c.GlobalString("home-dir")
				if home == "" {
					home = getHome()
				}
				_, _, _, err = dotfilesSN.Remove(session, home, c.Args(), c.Bool("quiet"), c.GlobalBool("debug"))
				if err != nil {
					return err
				}
				return err
			},
		},
	}
	sort.Sort(cli.FlagsByName(app.Flags))
	return msg, display, app.Run(args)
}

func getSession(server string) (gosn.Session, string, error) {
	var sess gosn.Session
	var email string
	var err error
	var password, apiServer, errMsg string
	email, password, apiServer, errMsg = dotfilesSN.GetCredentials(server)
	if errMsg != "" {
		fmt.Printf("\nerror: %s\n\n", errMsg)
		return sess, email, err
	}
	sess, err = gosn.CliSignIn(email, password, apiServer)
	if err != nil {
		return sess, email, err
	}
	return sess, email, err
}

func stripHome(in, home string) string {
	if home == "" {
		panic("home required")
	}
	if in == home {
		return ""
	}
	if strings.HasPrefix(in, home) {
		return in[len(home)+1:]
	}
	return in
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
	homeRelPath := stripHome(dir+filename, home)
	return strings.HasPrefix(homeRelPath, ".")
}
