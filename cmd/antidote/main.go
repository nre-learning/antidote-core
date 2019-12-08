package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	cli "github.com/urfave/cli"

	config "github.com/nre-learning/syringe/config"
	db "github.com/nre-learning/syringe/db"
)

func main() {

	app := cli.NewApp()
	app.Name = "antidote"
	// app.Version = buildInfo["buildVersion"]
	app.Version = "0.5.0"
	app.Usage = "Command-line tool to interact with the Antidote platform and database"

	// Consider build your own template that groups commands neatly
	// https://github.com/urfave/cli/blob/master/docs/v2/manual.md#customization-1
	// cli.AppHelpTemplate is where you do this, and cli.Command.Category can be used to organize

	app.Commands = []cli.Command{
		{
			Name:    "import",
			Aliases: []string{"import"},
			Usage:   "antidote import <CURRICULUM DIRECTORY>",
			Action: func(c *cli.Context) {

				color.Red("WARNING - This will DROP ALL DATA in the Antidote database.")
				fmt.Println("Are you sure you want to re-import a curriculum? (yes/no)")

				if !askForConfirmation() {
					os.Exit(0)
				}

				adb := db.AntidoteDB{
					User:     "postgres",
					Password: "docker",
					Database: "antidote",
					SyringeConfig: &config.SyringeConfig{
						// TODO(mierdin) Use a real syringeconfig
						Tier:          "local",
						CurriculumDir: c.Args().First(),
					},
				}

				// Initialize database
				// TODO(mierdin): Add confirmation, as this will drop all tables and recreate
				err := adb.Initialize()
				if err != nil {
					color.Red("Failed to initialize Antidote database.")
					fmt.Println(err)
					os.Exit(1)
				}

				// TODO(mierdin): Add collections, curriculum, meta
				// Collections should be first, so we can check that the collection exists in lesson import

				lessons, err := adb.ReadLessons()
				if err != nil {
					color.Red("Some curriculum resources failed to validate.")
					os.Exit(1)
				}

				err = adb.InsertLessons(lessons)
				if err != nil {
					color.Red("Problem inserting lessons into the database: %v", err)
					os.Exit(1)
				}

				color.Green("All detected curriculum resources imported successfully.")
				os.Exit(0)
			},
		},
		{
			Name:    "lesson",
			Aliases: []string{"lesson"},
			Usage:   "antidote lesson <subcommand>",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "List lessons",
					Action: func(c *cli.Context) {

						adb := db.AntidoteDB{
							User:          "postgres",
							Password:      "docker",
							Database:      "antidote",
							SyringeConfig: &config.SyringeConfig{},
						}

						lessons, err := adb.ListLessons()
						if err != nil {
							color.Red("Problem retrieving lessons: %v", err)
							os.Exit(1)
						}

						w := new(tabwriter.Writer)

						// Format in tab-separated columns with a tab stop of 8.

						w.Init(os.Stdout, 0, 8, 0, '\t', 0)

						fmt.Fprintln(w, "NAME\tSLUG\tCATEGORY\tTIER\t")

						for i := range lessons {
							l := lessons[i]
							fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t", l.LessonName, l.Slug, l.Category, l.Tier))
						}
						fmt.Fprintln(w)

						w.Flush()
					},
				},
				{
					Name:  "get",
					Usage: "Get a single lesson by slug reference",
					Action: func(c *cli.Context) {

						adb := db.AntidoteDB{
							User:          "postgres",
							Password:      "docker",
							Database:      "antidote",
							SyringeConfig: &config.SyringeConfig{},
						}

						lesson, err := adb.GetLesson(c.Args().First())
						if err != nil {
							color.Red("Problem retrieving lesson: %v", err)
							os.Exit(1)
						}

						b, err := json.Marshal(lesson)
						if err != nil {
							color.Red("Unable to print lesson details.")
							fmt.Println(err)
						}
						fmt.Println(string(b))
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
