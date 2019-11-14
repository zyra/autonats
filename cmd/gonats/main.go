package main

import (
	"github.com/urfave/cli"
	"github.com/zyra/autonats"
	"log"
	"os"
)

var AppVersion = "0.0.1"

func main() {
	app := cli.NewApp()
	app.Name = "Autonats"
	app.Version = AppVersion

	wd, _ := os.Getwd()

	app.Commands = []cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"g"},
			Usage:   "Generate NATS server handler + client files",
			Action: func(ctx *cli.Context) error {
				c := &autonats.ParserConfig{
					BaseDir:     ctx.String("dir"),
					NatsTimeout: ctx.Int("timeout"),
					OutFileName: ctx.String("out"),
				}

				if p, e := autonats.NewParser(c); e != nil {
					return e
				} else {
					p.Run()
				}

				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "dir, d",
					Usage:    "Base directory to search for matching interfaces",
					EnvVar:   "AUTONATS_BASE_DIR",
					Required: true,
					Value:    wd,
				},
				cli.IntFlag{
					Name:     "timeout, t",
					Usage:    "NATS request timeout in seconds",
					EnvVar:   "AUTONATS_REQUEST_TIMEOUT",
					Required: false,
					Value:    3,
				},
				cli.StringFlag{
					Name:   "out, o",
					Usage:  "Name to use for output file",
					EnvVar: "AUTONATS_OUT_FILE",
					Value:  "nats_client.go",
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
