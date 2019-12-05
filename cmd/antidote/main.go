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

	app.Commands = []cli.Command{
		{
			Name:    "import",
			Aliases: []string{"import"},
			Usage:   "antidote import <CURRICULUM DIRECTORY>",
			Action: func(c *cli.Context) {

				type AntidoteDB struct {
					User     string
					Password string
					Database string
				}

				adb := db.AntidoteDB{
					User:     "postgres",
					Password: "docker",
					Database: "antidote",
				}

				// TODO(mierdin): Add confirmation, as this will drop all tables and recreate

				// TODO(mierdin): Add collections, curriculum, meta
				// Collections should be first, so we can check that the collection exists in lesson import

				// TODO(mierdin) Use a real syringeconfig
				err := adb.ImportLessons(&config.SyringeConfig{
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
