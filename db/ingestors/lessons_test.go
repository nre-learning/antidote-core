package db

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	models "github.com/nre-learning/antidote-core/db/models"
)

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

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
				// Objectives: []*models.LessonStageObjective{
				// 	{
				// 		Description: "foobar",
				// 	},
				// },
			},
		},
		Name: "Example Lesson",
		Endpoints: []*models.LessonEndpoint{
			{
				Name:              "foobar1",
				Image:             "utility",
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
				Image:             "utility",
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
				Image:             "utility",
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
		Category: "fundamentals",
		Diagram:  "https://example.com/diagram.png",
		Video:    "https://example.com/video.png",
		Tier:     "local",
		Prereqs:  []string{},
		Tags:     []string{"a", "b", "c"},
		// Collection:  1,
		Description: "",

		// Path to mock lesson in the codebase (this is way better than mocking ioutil, IMO)
		LessonFile: "../test/mocklessons/validlesson1/lesson.meta.yaml",
	}
}

func TestValidLesson(t *testing.T) {
	l := getValidLesson()
	err := validateLesson(&l)
	assert(t, (err == nil), "Expected validation to pass, but encountered validation errors")
}

func TestInvalidCharInImageName(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Image = "antidotelabs/utility:latest"
	err := validateLesson(&l)

	assert(t, (err == BasicValidationError), "Expected a BasicValidationError")
}

// All Presentations must specify a nonzero TCP port
func TestMissingPresentationPort(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Presentations[0].Port = 0
	err := validateLesson(&l)

	assert(t, (err == BasicValidationError), "Expected a BasicValidationError")
}
