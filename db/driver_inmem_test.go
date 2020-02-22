package db

import (
	"testing"

	models "github.com/nre-learning/syringe/db/models"
)

func TestLessonsCRUD(t *testing.T) {

	adb := NewADMInMem()

	err := adb.InsertLessons([]*models.Lesson{
		{
			Name: "Foo Bar Lesson",
			Slug: "foobar",
		},

		// Intentionally repeating the last lesson, since the inmemory driver silently replaces
		// duplicates, and these tests assume that this behavior is in place and accounts for it.
		{
			Name: "Foo Bar Lesson Two",
			Slug: "foobar",
		},
		{
			Name: "Amazing Other Lesson",
			Slug: "amazing",
		},
	})
	if err != nil {
		t.Fatalf("Problem inserting lessons: %v", err)
	}

	lessons, err := adb.ListLessons()
	if err != nil {
		t.Fatalf("Problem listing lessons: %v", err)
	}
	if len(lessons) != 2 {
		t.Fatalf("Expected %d lessons, got %d", len(lessons), 2)
	}

	lesson, err := adb.GetLesson("foobar")
	if err != nil {
		t.Fatalf("Problem getting lessons: %v", err)
	}
	if lesson.Name != "Foo Bar Lesson Two" {
		t.Fatalf("Retrieved incorrect lesson: %s", lesson.Name)
	}

	_, err = adb.GetLesson("foobar2")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetLessons")
	}

}

func TestImagesCRUD(t *testing.T) {

	adb := NewADMInMem()

	err := adb.InsertImages([]*models.Image{
		{
			Slug:        "foobar",
			Description: "Foo Bar Image",
		},

		// Intentionally repeating the last image, since the inmemory driver silently replaces
		// duplicates, and these tests assume that this behavior is in place and accounts for it.
		{
			Slug:        "foobar",
			Description: "Foo Bar Image Two",
		},
		{
			Slug:        "amazing",
			Description: "Amazing Other Image",
		},
	})
	if err != nil {
		t.Fatalf("Problem inserting images: %v", err)
	}

	images, err := adb.ListImages()
	if err != nil {
		t.Fatalf("Problem listing images: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("Expected %d images, got %d", len(images), 2)
	}

	image, err := adb.GetImage("foobar")
	if err != nil {
		t.Fatalf("Problem getting images: %v", err)
	}
	if image.Description != "Foo Bar Image Two" {
		t.Fatalf("Retrieved incorrect image: (Got %s)", image.Description)
	}

	_, err = adb.GetImage("foobar2")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetImages")
	}

}

func TestCollectionsCRUD(t *testing.T) {

	adb := NewADMInMem()

	err := adb.InsertCollections([]*models.Collection{
		{
			Slug:  "foobar",
			Title: "Foo Bar Collection",
		},

		// Intentionally repeating the last collection, since the inmemory driver silently replaces
		// duplicates, and these tests assume that this behavior is in place and accounts for it.
		{
			Slug:  "foobar",
			Title: "Foo Bar Collection Two",
		},
		{
			Slug:  "amazing",
			Title: "Amazing Other Collection",
		},
	})
	if err != nil {
		t.Fatalf("Problem inserting collections: %v", err)
	}

	collections, err := adb.ListCollections()
	if err != nil {
		t.Fatalf("Problem listing collections: %v", err)
	}
	if len(collections) != 2 {
		t.Fatalf("Expected %d collections, got %d", len(collections), 2)
	}

	collection, err := adb.GetCollection("foobar")
	if err != nil {
		t.Fatalf("Problem getting collections: %v", err)
	}
	if collection.Title != "Foo Bar Collection Two" {
		t.Fatalf("Retrieved incorrect collection: (Got %s)", collection.Title)
	}

	_, err = adb.GetCollection("foobar2")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetCollection")
	}

}

func TestLiveLessonCRUD(t *testing.T) {

	adb := NewADMInMem()

	liveLessons := []*models.LiveLesson{
		{
			ID:          "10-abcdef",
			SessionID:   "abcdef",
			LessonID:    10,
			LessonStage: 1,
		}, {
			ID:          "11-abcdef",
			SessionID:   "abcdef",
			LessonID:    11,
			LessonStage: 1,
		}, {
			ID:          "10-ghijk",
			SessionID:   "ghijk",
			LessonID:    10,
			LessonStage: 1,
		},
	}

	for l := range liveLessons {
		err := adb.CreateLiveLesson(liveLessons[l])
		if err != nil {
			t.Fatalf("Problem creating LiveLesson: %v", err)
		}
	}

	err := adb.CreateLiveLesson(&models.LiveLesson{
		ID:        "10-ghijk",
		SessionID: "ghijk",
		LessonID:  10,
	})
	if err == nil {
		t.Fatal("Expected error creating LiveLesson but encountered none")
	}

	liveLessonsList, err := adb.ListLiveLessons()
	if err != nil {
		t.Fatalf("Problem listing LiveLessons: %v", err)
	}

	if len(liveLessonsList) != 3 {
		t.Fatalf("Expected %d liveLessons, got %d", len(liveLessonsList), 3)
	}

	ll, err := adb.GetLiveLesson("10-abcdef")
	if err != nil {
		t.Fatalf("Problem getting LiveLessons: %v", err)
	}
	if ll.SessionID != "abcdef" {
		t.Fatalf("Retrieved incorrect LiveLesson: (Got %s)", ll.SessionID)
	}

	_, err = adb.GetLiveLesson("foobar")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetLiveLesson")
	}

	ll.LessonStage = 2
	err = adb.UpdateLiveLesson(ll)
	if err != nil {
		t.Fatalf("Problem updating LiveLesson: %v", err)
	}
	newLl, err := adb.GetLiveLesson("10-abcdef")
	ok(t, err)
	assert(t, newLl.LessonStage == 2, "update check failed")

	err = adb.UpdateLiveLesson(&models.LiveLesson{
		ID:          "10-foobardoesntexist",
		SessionID:   "ghijk",
		LessonID:    10,
		LessonStage: 2,
	})
	if err == nil {
		t.Fatal("Error expected and was not produced in UpdateLiveLesson")
	}

	err = adb.DeleteLiveLesson("11-abcdef")
	if err != nil {
		t.Fatalf("Problem deleting LiveLesson: %v", err)
	}

	finalLiveLessonsList, err := adb.ListLiveLessons()
	ok(t, err)
	assert(t, len(finalLiveLessonsList) == 2, "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-abcdef"].LessonStage == 2, "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-abcdef"].SessionID == "abcdef", "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-ghijk"].LessonStage == 1, "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-ghijk"].SessionID == "ghijk", "final livelesson assertion failed")
}

func TestLiveSessionCRUD(t *testing.T) {

	adb := NewADMInMem()

	liveSessions := []*models.LiveSession{
		{
			ID:       "abcdef",
			SourceIP: "1.1.1.1",
		}, {
			ID:       "ghijkl",
			SourceIP: "2.2.2.2",
		}, {
			ID:       "mnopqr",
			SourceIP: "3.3.3.3",
		},
	}

	for s := range liveSessions {
		err := adb.CreateLiveSession(liveSessions[s])
		if err != nil {
			t.Fatalf("Problem creating LiveSession: %v", err)
		}
	}

	err := adb.CreateLiveSession(&models.LiveSession{
		ID:       "abcdef",
		SourceIP: "1.1.1.1",
	})
	if err == nil {
		t.Fatal("Expected error creating LiveSession but encountered none")
	}

	liveSessionsList, err := adb.ListLiveSessions()
	if err != nil {
		t.Fatalf("Problem listing LiveSessions: %v", err)
	}

	if len(liveSessionsList) != 3 {
		t.Fatalf("Expected %d liveSessions, got %d", len(liveSessionsList), 3)
	}

	ls, err := adb.GetLiveSession("abcdef")
	if err != nil {
		t.Fatalf("Problem getting LiveSessions: %v", err)
	}
	if ls.ID != "abcdef" {
		t.Fatalf("Retrieved incorrect LiveSession: (Got %s)", ls.ID)
	}

	_, err = adb.GetLiveSession("foobar")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetLiveSession")
	}

	ls.SourceIP = "123.123.123.123"
	err = adb.UpdateLiveSession(ls)
	if err != nil {
		t.Fatalf("Problem updating LiveSession: %v", err)
	}
	newLs, _ := adb.GetLiveSession("abcdef")
	assert(t, newLs.SourceIP == "123.123.123.123", "update check failed")

	err = adb.UpdateLiveSession(&models.LiveSession{
		ID:       "foobardoesntexist",
		SourceIP: "111.222.111.222",
	})
	if err == nil {
		t.Fatal("Error expected and was not produced in UpdateLiveSession")
	}

	err = adb.DeleteLiveSession("ghijkl")
	if err != nil {
		t.Fatalf("Problem deleting LiveSession: %v", err)
	}

	finalLiveSessionsList, _ := adb.ListLiveSessions()
	assert(t, len(finalLiveSessionsList) == 2, "final livesession assertion failed")
	assert(t, finalLiveSessionsList["abcdef"].SourceIP == "123.123.123.123", "final livesession assertion failed")
	assert(t, finalLiveSessionsList["abcdef"].ID == "abcdef", "final livesession assertion failed")
	assert(t, finalLiveSessionsList["mnopqr"].SourceIP == "3.3.3.3", "final livesession assertion failed")
	assert(t, finalLiveSessionsList["mnopqr"].ID == "mnopqr", "final livesession assertion failed")
}
