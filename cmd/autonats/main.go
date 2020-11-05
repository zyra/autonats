package main

import (
	"fmt"
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
				conc := ctx.Int("concurrency")

				if outFile == "" {
					outFile = "nats_client.go"
				} else if filepath.Ext(outFile) != ".go" {
					outFile += ".go"
				}

				if timeout <= 0 {
					timeout = 5
				}

				if conc <= 0 {
					conc = 5
				}

				fmt.Printf("parsing '%s' and will export to '%s'\n", baseDir, outFile)

				parser := autonats.NewParser(&autonats.ParserConfig{
					BaseDir:            baseDir,
					DefaultTimeout:     timeout,
					OutputFileName:     outFile,
					DefaultConcurrency: conc,
					Tracing:            ctx.Bool("tracing"),
				})

				if err := parser.ParseDir(baseDir); err != nil {
					return fmt.Errorf("failed to parse the provided directory: %s", err.Error())
				}

				parser.Run()

				return parser.Render()
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
				cli.BoolFlag{
					Name:   "tracing",
					Usage:  "Generate tracing code using OpenTracing library",
					EnvVar: "AUTONATS_TRACING",
				},
				cli.IntFlag{
					Name:   "concurrency, c",
					Usage:  "Default handler concurrency",
					EnvVar: "AUTONATS_CONCURRENCY",
					Value:  5,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
