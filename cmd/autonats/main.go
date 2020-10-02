package main

import (
	"github.com/urfave/cli"
	"github.com/zyra/autonats"
	"log"
	"os"
	"path/filepath"
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
				baseDir := ctx.String("dir")
				timeout := ctx.Int("timeout")
				outFile := ctx.String("out")

				if outFile == "" {
					outFile = "nats_client.go"
				} else if filepath.Ext(outFile) != ".go" {
					outFile += ".go"
				}

				if timeout <= 0 {
					timeout = 5
				}

				parser := autonats.NewParser()

				if err := parser.ParseDir(baseDir); err != nil {
					return err
				}

				parser.Run()

				parser.Render(baseDir, outFile, timeout)

				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "dir, d",
					Usage:  "Base directory to search for matching interfaces",
					EnvVar: "AUTONATS_BASE_DIR",
					Value:  wd,
				},
				cli.IntFlag{
					Name:   "timeout, t",
					Usage:  "NATS request timeout in seconds",
					EnvVar: "AUTONATS_REQUEST_TIMEOUT",
					Value:  5,
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
