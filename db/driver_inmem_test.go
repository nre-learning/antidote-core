package db

import (
	"testing"

	models "github.com/nre-learning/syringe/db/models"
)

func TestLessonCRUD(t *testing.T) {

	adb := NewADMInMem()

	lessons := []*models.Lesson{
		{
			Name: "foobar",
			Slug: "foobar",
		},
	}

	err := adb.InsertLessons(lessons)
	if err != nil {
		t.Fatalf("Problem inserting lessons into the database: %v", err)
	}
}
