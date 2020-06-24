package main

import (
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/urfave/cli/v2"
)

var sp *spinner.Spinner

func main() {
	sp = spinner.New(spinner.CharSets[70], 100*time.Millisecond)

	app := cli.NewApp()
	app.Name = "synchub"
	app.Usage = "keep github in sync!"
	app.Commands = commands()
	app.Flags = flags()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:            "sync",
			SkipFlagParsing: false,
			Aliases:         []string{"s"},
			Usage:           "specify yml file(s) to sync",
			Action:          runSync,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "confirm-public",
					Usage: "ask confirmation when repository is public",
				},
				&cli.StringFlag{
					Name:    "repos",
					Usage:   "restrict sync to one or more defined repo, separated by commas",
					Aliases: []string{"r"},
				},
			},
		},
	}
}

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"vvv"},
		},
		&cli.StringFlag{
			Name:     "token",
			Usage:    "github token",
			Required: true,
			EnvVars:  []string{"GITHUB_TOKEN"},
			Aliases:  []string{"t"},
		},
	}
}

func runSync(c *cli.Context) error {
	sync := Sync{
		files:         c.Args().Slice(),
		token:         c.String("token"),
		repos:         c.String("repos"),
		verbose:       c.Bool("verbose"),
		confirmPublic: c.Bool("confirm-public"),
	}
	sync.Exec()
	return nil
}
