package main

import (
	"encoding/json"
	"os"

	"github.com/fatih/color"
	cli "github.com/urfave/cli"

	ingestors "github.com/nre-learning/antidote-core/db/ingestors"
	models "github.com/nre-learning/antidote-core/db/models"
)

func main() {

	app := cli.NewApp()
	app.Name = "antidote"
	app.Version = buildInfo["buildVersion"]
	app.Usage = "Companion tool for working with Antidote-based curricula"

	var curriculumDir string

	app.Commands = []cli.Command{
		{
			Name:    "validate",
			Aliases: []string{},
			Usage:   "Validates a full curriculum directory for correctness",
			Action: func(c *cli.Context) {

				_, err := ingestors.ReadImages(c.Args().First())
				if err != nil {
					color.Red("Some curriculum resources failed to validate.")
					os.Exit(1)
				}

				_, err = ingestors.ReadCollections(c.Args().First())
				if err != nil {
					color.Red("Some curriculum resources failed to validate.")
					os.Exit(1)
				}

				_, err = ingestors.ReadLessons(c.Args().First())
				if err != nil {
					color.Red("Some curriculum resources failed to validate.")
					os.Exit(1)
				}

				color.Green("All detected curriculum resources imported successfully.")
			},
		},
		{
			Name:    "lesson",
			Aliases: []string{},
			Usage:   "Work with Lesson resources",
			Subcommands: []cli.Command{
				{
					Name:  "create",
					Usage: "Create a lesson using an interactive wizard",

					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "C, curriculum-directory",
							Usage:       "antidote lesson create -L ./",
							Value:       ".",
							Destination: &curriculumDir,
						},
					},

					Action: func(c *cli.Context) {
						color.Green("Interactively creating new Lesson (https://docs.nrelabs.io/antidote/object-reference/lessons)")

						// Create blank Lesson instance and associated schema
						newLesson := models.Lesson{}
						lessonSchema := newLesson.GetSchema()

						// Start interactive creation wizard
						lessonData, err := schemaWizard(lessonSchema, "Lesson", "")

						// The schema wizard returns a string-indexed map, so we want to marshal
						// the full result to JSON, and then into the Lesson type
						stmJSON, err := json.Marshal(lessonData)
						if err != nil {
							panic(err)
						}
						json.Unmarshal([]byte(stmJSON), &newLesson)

						// Pass populated lesson definition to the rendering function
						renderLessonFiles(curriculumDir, &newLesson)
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
