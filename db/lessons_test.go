package db

import (
	"testing"

	config "github.com/nre-learning/syringe/config"
	models "github.com/nre-learning/syringe/db/models"
)

// getValidLesson returns a full, valid example of a Lesson that uses all the features.
// Tests in this file should make use of this by making a copy, tweaking in some way that makes it
// invalid, and then asserting on the error type/message.
func getValidLesson() models.Lesson {
	return models.Lesson{
		Slug: "example-lesson",
		Stages: []*models.LessonStage{
			{
				Description: "Test Stage",
				GuideType:   "markdown",
				Objectives: []*models.LessonStageObjective{
					{
						Description: "foobar",
					},
				},
			},
		},
		LessonName: "Example Lesson",
		Endpoints: []*models.LessonEndpoint{
			{
				Name:              "foobar1",
				Image:             "antidotelabs/utility",
				ConfigurationType: "napalm-junos",
				Presentations: []*models.LessonPresentation{
					{
						Name: "presentation1",
						Port: 22,
						Type: "http",
					},
					{
						Name: "presentation2",
						Port: 80,
						Type: "ssh",
					},
				},
			},
			{
				Name:              "foobar2",
				Image:             "antidotelabs/utility",
				ConfigurationType: "python",
				Presentations: []*models.LessonPresentation{
					{
						Name: "presentation1",
						Port: 22,
						Type: "http",
					},
					{
						Name: "presentation2",
						Port: 80,
						Type: "ssh",
					},
				},
			},
			{
				Name:              "foobar3",
				Image:             "antidotelabs/utility",
				ConfigurationType: "ansible",
				Presentations: []*models.LessonPresentation{
					{
						Name: "presentation1",
						Port: 22,
						Type: "http",
					},
					{
						Name: "presentation2",
						Port: 80,
						Type: "ssh",
					},
				},
			},
		},
		Connections: []*models.LessonConnection{
			{
				A: "foobar1",
				B: "foobar2",
			},
			{
				A: "foobar2",
				B: "foobar3",
			},
			{
				A: "foobar3",
				B: "foobar1",
			},
		},
		Category:      "fundamentals",
		LessonDiagram: "https://example.com/diagram.png",
		LessonVideo:   "https://example.com/video.png",
		Tier:          "local",
		Prereqs:       []string{},
		Tags:          []string{"a", "b", "c"},
		Collection:    1,
		Description:   "",

		// Path to mock lesson in the codebase (this is way better than mocking ioutil, IMO)
		LessonFile: "test/mocklessons/validlesson1/lesson.meta.yaml",
	}
}

func TestValidLesson(t *testing.T) {
	l := getValidLesson()
	err := validateLesson(&config.SyringeConfig{
		Tier: "local",
	}, &l)
	assert(t, (err == nil), "Expected validation to pass, but encountered validation errors")
}

// test invalid image name
func TestImageName(t *testing.T) {
	l := getValidLesson()
	// colons not allowed
	l.Endpoints[0].Image = "antidotelabs/utility:latest"
	err := validateLesson(&config.SyringeConfig{
		Tier: "local",
	}, &l)

	assert(t, (err == BasicValidationError), "Expected a BasicValidationError")
}

// All Presentations must specify a nonzero TCP port
func TestMissingPresentationPort(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Presentations[0].Port = 0
	err := validateLesson(&config.SyringeConfig{
		Tier: "local",
	}, &l)

	assert(t, (err == BasicValidationError), "Expected a BasicValidationError")
}
