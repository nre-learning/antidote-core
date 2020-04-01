package db

import (
	"testing"

	"github.com/nre-learning/antidote-core/config"
	models "github.com/nre-learning/antidote-core/db/models"
)

// getValidLesson returns a full, valid example of a Lesson that uses all the features.
// Tests in this file should make use of this by making a copy, tweaking in some way that makes it
// invalid, and then asserting on the error type/message.
func getValidLesson() models.Lesson {

	lessons, err := ReadLessons(config.AntidoteConfig{
		CurriculumDir: "../test/test-curriculum",
		Tier:          "local",
	})
	if err != nil {
		panic(err)
	}

	for _, l := range lessons {
		if l.Slug == "valid-lesson" {
			return *l
		}
	}
	panic("unable to find valid lesson")
}

func TestValidLesson(t *testing.T) {
	l := getValidLesson()
	err := validateLesson(&l)
	assert(t, (err == nil), "Expected validation to pass, but encountered validation errors")
}

func TestBadLessonSlug(t *testing.T) {
	l := getValidLesson()
	l.Slug = "foobar9b3#$(*#"
	err := validateLesson(&l)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}

func TestInvalidCharInImageName(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Image = "antidotelabs/utility:latest"
	err := validateLesson(&l)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}

// All Presentations must specify a nonzero TCP port
func TestMissingPresentationPort(t *testing.T) {
	l := getValidLesson()
	l.Endpoints[0].Presentations[0].Port = 0
	err := validateLesson(&l)

	assert(t, (err == errBasicValidation), "Expected errBasicValidation")
}
