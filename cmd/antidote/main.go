package main

import (
	"encoding/json"
	"os"

	"github.com/fatih/color"
	cli "github.com/urfave/cli"

	ingestors "github.com/nre-learning/syringe/db/ingestors"
	models "github.com/nre-learning/syringe/db/models"
)

func main() {

	app := cli.NewApp()
	app.Name = "antidote"
	app.Version = buildInfo["buildVersion"]
	app.Usage = "Command-line tool to interact with the Antidote platform and database"

	// Consider build your own template that groups commands neatly
	// https://github.com/urfave/cli/blob/master/docs/v2/manual.md#customization-1
	// cli.AppHelpTemplate is where you do this, and cli.Command.Category can be used to organize

	var curriculumDir string

	testLesson := models.Lesson{
		Slug:     "test-lesson",
		Name:     "Test Lesson",
		Category: "Fundamentals",
		Tier:     "prod",
		Endpoints: []*models.LessonEndpoint{{
			Name:  "linux1",
			Image: "antidotelabs/utility",
			Presentations: []*models.LessonPresentation{{
				Name: "cli",
				Type: "ssh",
				Port: 22,
			}},
		}},
		Stages: []*models.LessonStage{
			{
				Description: "stage0",
				GuideType:   "markdown",
			},
			{
				Description: "stage1",
				GuideType:   "jupyter",
			},
		},
		Connections: []*models.LessonConnection{},
	}

	app.Commands = []cli.Command{
		{
			Name:    "validate",
			Aliases: []string{},
			Usage:   "Validates a full curriculum directory for correctness",
			Action: func(c *cli.Context) {

				_, err := ingestors.ReadLessons(c.Args().First())
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

						// Interactively populate fields in Lesson
						newLesson := models.Lesson{}
						lessonSchema := newLesson.GetSchema()
						lessonData, err := schemaWizard(lessonSchema, "Lesson", "")

						// Marshal map to JSON and then unmarshal JSON to Lesson type
						stmJSON, err := json.Marshal(lessonData)
						if err != nil {
							panic(err)
						}
						json.Unmarshal([]byte(stmJSON), &newLesson)

						// Pass populated lesson definition to the rendering function
						// renderLessonFiles(curriculumDir, &newLesson)
						renderLessonFiles(curriculumDir, &testLesson)

					},
				},
			},
		},
	}

	app.Run(os.Args)
}
