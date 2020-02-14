package db

import (
	"testing"
)

func TestLessonCRUD(t *testing.T) {

	err = adb.InsertLessons(lessons)
	if err != nil {
		t.Fatalf("Problem inserting lessons into the database: %v", err)
	}
}
