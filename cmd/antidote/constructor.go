package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	models "github.com/nre-learning/syringe/db/models"
	"gopkg.in/yaml.v2"
)

// TODO(mierdin): When finished, produce a list of things that the user will still have to do themselves.
// Or, perhaps just run the validation logic once the files are rendered?

// TODO this function needs to do some good, colorized logging of what its creating, and then what is left to do
func renderLessonFiles(curriculumDir string, lesson *models.Lesson) error {

	// Set lesson directory
	lessonDir := fmt.Sprintf("%s/lessons/%s", curriculumDir, lesson.Slug)
	askSimpleValue("Please enter location for lesson directory, or press enter to accept default", lessonDir)

	// TODO what if this already exists?
	err := os.MkdirAll(lessonDir, os.ModePerm)
	if err != nil {
		return err
	}
	color.Green("--- Created lesson directory %s", lessonDir)

	yamlOutput, err := yaml.Marshal(&lesson)
	if err != nil {
		color.Red("Unable to print lesson details.")
		fmt.Println(err)
	}
	color.Green("Using %s to store created lesson definitions\n", lessonDir)

	fmt.Printf("---\n%s\n\n", string(yamlOutput))

	return nil
}
