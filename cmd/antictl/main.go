package main

import (
	"os"

	cli "github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "antidote"
	app.Version = buildInfo["buildVersion"]
	app.Usage = "Command-line tool to interact with the Antidote platform and database"

	// global flags
	var dbuser, dbpassword string
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "H, db-username",
			Usage:       "Database username",
			Value:       "postgres",
			Destination: &dbuser,
		},
		cli.StringFlag{
			Name:        "P, db-password",
			Usage:       "Database password",
			Value:       "docker",
			Destination: &dbpassword,
		},
	}

	// Consider build your own template that groups commands neatly
	// https://github.com/urfave/cli/blob/master/docs/v2/manual.md#customization-1
	// cli.AppHelpTemplate is where you do this, and cli.Command.Category can be used to organize

	// var curriculumDir string

	app.Commands = []cli.Command{
		{
			Name:    "import",
			Aliases: []string{},
			Usage:   "(db) Re-imports a full curriculum from disk",
			Action: func(c *cli.Context) {

				// color.Red("WARNING - This will DROP ALL DATA in the Antidote database.")
				// fmt.Println("Are you sure you want to re-import a curriculum? (yes/no)")

				// if !askForConfirmation() {
				// 	os.Exit(0)
				// }

				// adb := db.AntidoteData{
				// 	User:            dbuser,
				// 	Password:        dbpassword,
				// 	Database:        "antidote",
				// 	AntidoteVersion: buildInfo["buildVersion"],
				// 	// Connect:         prodConnect(),
				// 	SyringeConfig: &config.SyringeConfig{
				// 		// TODO(mierdin) Use a real syringeconfig
				// 		Tier:          "local",
				// 		CurriculumDir: c.Args().First(),
				// 	},
				// }

				// // Initialize database
				// err := adb.Initialize()
				// if err != nil {
				// 	color.Red("Failed to initialize Antidote database.")
				// 	fmt.Println(err)
				// 	os.Exit(1)
				// }

				// // TODO(mierdin): Add collections, curriculum, meta
				// // Collections should be first, so we can check that the collection exists in lesson import

				// lessons, err := adb.ReadLessons()
				// if err != nil {
				// 	color.Red("Some curriculum resources failed to validate.")
				// 	os.Exit(1)
				// }

				// err = adb.InsertLessons(lessons)
				// if err != nil {
				// 	color.Red("Problem inserting lessons into the database: %v", err)
				// 	os.Exit(1)
				// }

				// color.Green("All detected curriculum resources imported successfully.")
				// os.Exit(0)
			},
		},
		{
			Name:    "lesson",
			Aliases: []string{},
			Usage:   "Work with Lesson resources",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "Retrieve all lessons and display in a table",
					Action: func(c *cli.Context) {

						// w := new(tabwriter.Writer)

						// // Format in tab-separated columns with a tab stop of 8.

						// w.Init(os.Stdout, 0, 8, 0, '\t', 0)

						// fmt.Fprintln(w, "NAME\tSLUG\tCATEGORY\tTIER\t")

						// for i := range lessons {
						// 	l := lessons[i]
						// 	fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t", l.Name, l.Slug, l.Category, l.Tier))
						// }
						// fmt.Fprintln(w)

						// w.Flush()
					},
				},
				{
					Name:  "get",
					Usage: "Show details of a specific lesson",
					Action: func(c *cli.Context) {

						// adb := db.AntidoteData{
						// 	User:            dbuser,
						// 	Password:        dbpassword,
						// 	Database:        "antidote",
						// 	AntidoteVersion: buildInfo["buildVersion"],
						// 	SyringeConfig:   &config.SyringeConfig{},
						// }

						// err := adb.Preflight()
						// if err != nil {
						// 	color.Red("Failed pre-flight.")
						// 	fmt.Println(err)
						// 	os.Exit(1)
						// }

						// lesson, err := adb.GetLesson(c.Args().First())
						// if err != nil {
						// 	color.Red("Problem retrieving lesson: %v", err)
						// 	os.Exit(1)
						// }

						// b, err := json.Marshal(lesson)
						// if err != nil {
						// 	color.Red("Unable to print lesson details.")
						// 	fmt.Println(err)
						// }
						// fmt.Println(string(b))
					},
				},
			},
		},
		{
			Name:    "livelesson",
			Aliases: []string{"ll"},
			Usage:   "Examine/modify running LiveLessons",
			Subcommands: []cli.Command{
				{
					Name:  "kill",
					Usage: "Kill a running livelesson",
					Action: func(c *cli.Context) {
						return // TODO
					},
				},
				{
					Name:  "list",
					Usage: "List all livelessons",
					Action: func(c *cli.Context) {
						return // TODO
					},
				},
				{
					Name:  "get",
					Usage: "Get details for a specific livelessons",
					Action: func(c *cli.Context) {
						return // TODO
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
