package main

import (
	"encoding/json"
	"fmt"
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

	app.Commands = []cli.Command{
		{
			Name:        "validate",
			Aliases:     []string{},
			Usage:       "Validates a full curriculum directory for correctness",
			UsageText:   "antidote validate <path>",
			Description: "Validates a full curriculum directory for correctness. Curriculum directory defaults to current working directory",
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
						err = renderLessonFiles(&newLesson)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					},
				},
			},
		},
		{
			Name:    "collection",
			Aliases: []string{},
			Usage:   "Work with Collection resources",
			Subcommands: []cli.Command{
				{
					Name:  "create",
					Usage: "Create a collection using an interactive wizard",

					Action: func(c *cli.Context) {
						color.Green("Interactively creating new Collection (https://docs.nrelabs.io/antidote/object-reference/collections)")

						// Create blank Collection instance and associated schema
						newCollection := models.Collection{}
						collectionSchema := newCollection.GetSchema()

						// Start interactive creation wizard
						collectionData, err := schemaWizard(collectionSchema, "Collection", "")

						// The schema wizard returns a string-indexed map, so we want to marshal
						// the full result to JSON, and then into the collection type
						stmJSON, err := json.Marshal(collectionData)
						if err != nil {
							panic(err)
						}
						json.Unmarshal([]byte(stmJSON), &newCollection)

						// Pass populated collection definition to the rendering function
						err = renderCollectionFiles(&newCollection)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
