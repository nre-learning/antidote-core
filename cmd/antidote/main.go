package main

import (
	"os"

	"github.com/fatih/color"
	cli "github.com/urfave/cli"

	config "github.com/nre-learning/syringe/config"
	db "github.com/nre-learning/syringe/db"
)

func main() {

	app := cli.NewApp()
	app.Name = "Antidote"
	// app.Version = buildInfo["buildVersion"]
	app.Usage = "Command-line tool to interact with Antidote"

	var host, port string

	// global level flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "H, host",
			Usage:       "syringed hostname",
			Value:       "127.0.0.1",
			Destination: &host,
		},
		cli.StringFlag{
			Name:        "P, port",
			Usage:       "syringed port",
			Value:       "50099",
			Destination: &port,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "import",
			Aliases: []string{"import"},
			Usage:   "antidote import <CURRICULUM DIRECTORY>",
			Action: func(c *cli.Context) {

				_, err := db.ImportCurriculum(&config.SyringeConfig{
					Tier:          "local",
					CurriculumDir: c.Args().First(),
				})
				if err != nil {
					color.Red("Some curriculum resources failed to validate.")
					os.Exit(1)
				} else {
					color.Green("All detected curriculum resources imported successfully.")
					os.Exit(0)
				}

			},
		},
	}

	app.Run(os.Args)
}
