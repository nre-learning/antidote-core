package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	models "github.com/nre-learning/antidote-core/db/models"
	"gopkg.in/yaml.v2"
)

const (
	defaultMarkdownContents = `
Please enter your lesson guide contents here.
`
	defaultJupyterContents = `
{
 "nbformat": 4,
 "nbformat_minor": 2,
 "metadata": {
  "language_info": {
   "name": "python",
   "codemirror_mode": {
    "name": "ipython",
    "version": 3
   }
  },
  "orig_nbformat": 2,
  "file_extension": ".py",
  "mimetype": "text/x-python",
  "name": "python",
  "npconvert_exporter": "python",
  "pygments_lexer": "ipython3",
  "version": 3
 },
 "cells": [
  {
   "cell_type": "code",
   "execution_count": null,
   "metadata": {},
   "outputs": [],
   "source": [
    "Please edit this jupyter notebook to suit your needs."
   ]
  }
 ]
}
`

	defaultAnsibleContents = `
Please replace this text with an Ansible playbook for configuring this endpoint.
`

	defaultPythonContents = `
Please replace this text with a Python script for configuring this endpoint.
`

	defaultNapalmContents = `
Please replace this text with a NAPALM-compatible configuration for this endpoint.

Also, don't forget to update the name of the file to include the desired NAPALM driver (i.e. "junos", "ios", etc.)

`
)

func renderLessonFiles(curriculumDir string, lesson *models.Lesson) error {

	// Set lesson directory
	lessonDir := fmt.Sprintf("%s/lessons/%s", curriculumDir, lesson.Slug)

	for {
		lessonDir = askSimpleValue("Where should I place this lesson? ", lessonDir)
		if _, err := os.Stat(lessonDir); os.IsNotExist(err) {
			break
		}
		color.Red("Location already exists. Please select another location.")
	}

	color.Green("--- ** WRITING SKELETON LESSON TO DISK **")

	err := os.MkdirAll(lessonDir, os.ModePerm)
	if err != nil {
		return err
	}
	color.Green("--- Created lesson directory %s", lessonDir)

	yamlOutput, err := yaml.Marshal(&lesson)
	if err != nil {
		color.Red("Unable to convert lesson to YAML.")
		fmt.Println(err)
		return err
	}

	meta := fmt.Sprintf("%s/lesson.meta.yaml", lessonDir)
	err = writeToFile(meta, string(yamlOutput))
	if err != nil {
		return err
	}

	color.Green("--- Created lesson metadata file %s", meta)

	for s := range lesson.Stages {
		stage := lesson.Stages[s]

		stageDirectory := fmt.Sprintf("%s/stage%d", lessonDir, s)

		err := os.MkdirAll(stageDirectory, os.ModePerm)
		if err != nil {
			return err
		}
		color.Green("--- Created stage directory %s", stageDirectory)

		var fileContents string
		var fileLocation string
		switch stage.GuideType {
		case "jupyter":
			fileContents = defaultJupyterContents
			fileLocation = fmt.Sprintf("%s/guide.ipynb", stageDirectory)
		default:
			fileContents = defaultMarkdownContents
			fileLocation = fmt.Sprintf("%s/guide.md", stageDirectory)
		}

		err = writeToFile(fileLocation, fileContents)
		if err != nil {
			return err
		}

		color.Green("--- Created lesson guide %s", fileLocation)

		configsDirectory := fmt.Sprintf("%s/configs", stageDirectory)
		err = os.MkdirAll(configsDirectory, os.ModePerm)
		if err != nil {
			return err
		}
		color.Green("--- Created configs directory %s", configsDirectory)
		for e := range lesson.Endpoints {
			ep := lesson.Endpoints[e]

			var fileContents string
			var fileLocation string
			switch stage.GuideType {
			case "ansible":
				fileContents = defaultAnsibleContents
				fileLocation = fmt.Sprintf("%s/%s.yaml", configsDirectory, ep.Name)
			case "python":
				fileContents = defaultPythonContents
				fileLocation = fmt.Sprintf("%s/%s.py", configsDirectory, ep.Name)
			case "":
				continue
			default:
				fileContents = defaultNapalmContents
				fileLocation = fmt.Sprintf("%s/%s-<napalm-driver-here>.txt", configsDirectory, ep.Name)
			}

			err = writeToFile(fileLocation, fileContents)
			if err != nil {
				return err
			}

			color.Green("--- Created config %s", fileLocation)
		}
	}

	fmt.Println("")
	color.Yellow("NOTE: This is just a skeleton lesson. There's still a lot more to do! For instance:")
	color.Yellow("- Use 'antidote validate' to identify anything you need to update/fix")
	color.Yellow("- Edit all configs in the 'configs/' directory of each stage to properly configure your endpoints")
	color.Yellow("- Write your content! All stage lesson guides are empty and waiting for your knowledge.")
	color.Yellow("- Open a Pull Request and Preview your Content! https://docs.nrelabs.io/creating-contributing/contributing-content")

	return nil
}

func writeToFile(location, contents string) error {
	f, err := os.Create(location)
	if err != nil {
		color.Red(fmt.Sprintf("%s - %s", location, err.Error()))
		return err
	}
	defer f.Close()

	_, err = f.WriteString(contents)
	if err != nil {
		color.Red(fmt.Sprintf("%s - %s", location, err.Error()))
		return err
	}

	return nil
}
