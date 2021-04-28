package db

import (
	"testing"

	models "github.com/nre-learning/antidote-core/db/models"
	ot "github.com/opentracing/opentracing-go"
)

func TestLessonsCRUD(t *testing.T) {

	span := ot.StartSpan("test_db")
	defer span.Finish()

	adb := NewADMInMem()

	err := adb.InsertLessons(span.Context(), []*models.Lesson{
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

	lessons, err := adb.ListLessons(span.Context())
	if err != nil {
		t.Fatalf("Problem listing lessons: %v", err)
	}
	if len(lessons) != 2 {
		t.Fatalf("Expected %d lessons, got %d", len(lessons), 2)
	}

	lesson, err := adb.GetLesson(span.Context(), "foobar")
	if err != nil {
		t.Fatalf("Problem getting lessons: %v", err)
	}
	if lesson.Name != "Foo Bar Lesson Two" {
		t.Fatalf("Retrieved incorrect lesson: %s", lesson.Name)
	}

	_, err = adb.GetLesson(span.Context(), "foobar2")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetLessons")
	}

}

func TestImagesCRUD(t *testing.T) {

	span := ot.StartSpan("test_db")
	defer span.Finish()

	adb := NewADMInMem()

	err := adb.InsertImages(span.Context(), []*models.Image{
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

	images, err := adb.ListImages(span.Context())
	if err != nil {
		t.Fatalf("Problem listing images: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("Expected %d images, got %d", len(images), 2)
	}

	image, err := adb.GetImage(span.Context(), "foobar")
	if err != nil {
		t.Fatalf("Problem getting images: %v", err)
	}
	if image.Description != "Foo Bar Image Two" {
		t.Fatalf("Retrieved incorrect image: (Got %s)", image.Description)
	}

	_, err = adb.GetImage(span.Context(), "foobar2")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetImages")
	}

}

func TestCollectionsCRUD(t *testing.T) {
	span := ot.StartSpan("test_db")
	defer span.Finish()

	adb := NewADMInMem()

	err := adb.InsertCollections(span.Context(), []*models.Collection{
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

	collections, err := adb.ListCollections(span.Context())
	if err != nil {
		t.Fatalf("Problem listing collections: %v", err)
	}
	if len(collections) != 2 {
		t.Fatalf("Expected %d collections, got %d", len(collections), 2)
	}

	collection, err := adb.GetCollection(span.Context(), "foobar")
	if err != nil {
		t.Fatalf("Problem getting collections: %v", err)
	}
	if collection.Title != "Foo Bar Collection Two" {
		t.Fatalf("Retrieved incorrect collection: (Got %s)", collection.Title)
	}

	_, err = adb.GetCollection(span.Context(), "foobar2")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetCollection")
	}

}

func TestLiveLessonCRUD(t *testing.T) {
	span := ot.StartSpan("test_db")
	defer span.Finish()
	adb := NewADMInMem()

	liveLessons := []*models.LiveLesson{
		{
			ID:           "10-abcdef",
			SessionID:    "abcdef",
			LessonSlug:   "foobar-10",
			CurrentStage: 0,
		}, {
			ID:           "11-abcdef",
			SessionID:    "abcdef",
			LessonSlug:   "foobar-11",
			CurrentStage: 0,
		}, {
			ID:           "10-ghijk",
			SessionID:    "ghijk",
			LessonSlug:   "foobar-10",
			CurrentStage: 1,
		},
	}

	for l := range liveLessons {
		err := adb.CreateLiveLesson(span.Context(), liveLessons[l])
		if err != nil {
			t.Fatalf("Problem creating LiveLesson: %v", err)
		}
	}

	err := adb.CreateLiveLesson(span.Context(), &models.LiveLesson{
		ID:         "10-ghijk",
		SessionID:  "ghijk",
		LessonSlug: "foobar-10",
	})
	if err == nil {
		t.Fatal("Expected error creating LiveLesson but encountered none")
	}

	liveLessonsList, err := adb.ListLiveLessons(span.Context())
	if err != nil {
		t.Fatalf("Problem listing LiveLessons: %v", err)
	}

	if len(liveLessonsList) != 3 {
		t.Fatalf("Expected %d liveLessons, got %d", len(liveLessonsList), 3)
	}

	ll, err := adb.GetLiveLesson(span.Context(), "10-abcdef")
	if err != nil {
		t.Fatalf("Problem getting LiveLessons: %v", err)
	}
	if ll.SessionID != "abcdef" {
		t.Fatalf("Retrieved incorrect LiveLesson: (Got %s)", ll.SessionID)
	}

	_, err = adb.GetLiveLesson(span.Context(), "foobar")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetLiveLesson")
	}

	err = adb.UpdateLiveLessonStage(span.Context(), ll.ID, 1)
	if err != nil {
		t.Fatalf("Problem updating LiveLesson: %v", err)
	}
	newLl, err := adb.GetLiveLesson(span.Context(), ll.ID)
	ok(t, err)
	assert(t, newLl.CurrentStage == 1, "update check failed")

	err = adb.UpdateLiveLessonStage(span.Context(), "10-foobardoesntexist", 2)
	if err == nil {
		t.Fatal("Error expected and was not produced in UpdateLiveLesson")
	}

	err = adb.DeleteLiveLesson(span.Context(), "11-abcdef")
	if err != nil {
		t.Fatalf("Problem deleting LiveLesson: %v", err)
	}

	finalLiveLessonsList, err := adb.ListLiveLessons(span.Context())
	ok(t, err)
	assert(t, len(finalLiveLessonsList) == 2, "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-abcdef"].CurrentStage == 1, "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-abcdef"].SessionID == "abcdef", "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-ghijk"].CurrentStage == 1, "final livelesson assertion failed")
	assert(t, finalLiveLessonsList["10-ghijk"].SessionID == "ghijk", "final livelesson assertion failed")
}

func TestLiveSessionCRUD(t *testing.T) {
	span := ot.StartSpan("test_db")
	defer span.Finish()
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
		err := adb.CreateLiveSession(span.Context(), liveSessions[s])
		if err != nil {
			t.Fatalf("Problem creating LiveSession: %v", err)
		}
	}

	err := adb.CreateLiveSession(span.Context(), &models.LiveSession{
		ID:       "abcdef",
		SourceIP: "1.1.1.1",
	})
	if err == nil {
		t.Fatal("Expected error creating LiveSession but encountered none")
	}

	liveSessionsList, err := adb.ListLiveSessions(span.Context())
	if err != nil {
		t.Fatalf("Problem listing LiveSessions: %v", err)
	}

	if len(liveSessionsList) != 3 {
		t.Fatalf("Expected %d liveSessions, got %d", 3, len(liveSessionsList))
	}

	ls, err := adb.GetLiveSession(span.Context(), "abcdef")
	if err != nil {
		t.Fatalf("Problem getting LiveSessions: %v", err)
	}
	if ls.ID != "abcdef" {
		t.Fatalf("Retrieved incorrect LiveSession: (Got %s)", ls.ID)
	}

	_, err = adb.GetLiveSession(span.Context(), "foobar")
	if err == nil {
		t.Fatal("Error expected and was not produced in GetLiveSession")
	}

	err = adb.DeleteLiveSession(span.Context(), "ghijkl")
	if err != nil {
		t.Fatalf("Problem deleting LiveSession: %v", err)
	}

	finalLiveSessionsList, _ := adb.ListLiveSessions(span.Context())
	assert(t, len(finalLiveSessionsList) == 2, "final livesession assertion failed")
	assert(t, finalLiveSessionsList["abcdef"].SourceIP == "1.1.1.1", "final livesession assertion failed")
	assert(t, finalLiveSessionsList["abcdef"].ID == "abcdef", "final livesession assertion failed")
	assert(t, finalLiveSessionsList["mnopqr"].SourceIP == "3.3.3.3", "final livesession assertion failed")
	assert(t, finalLiveSessionsList["mnopqr"].ID == "mnopqr", "final livesession assertion failed")
}

func TestGetLiveLessonsForSession(t *testing.T) {
	span := ot.StartSpan("test_db")
	defer span.Finish()

	adb := NewADMInMem()

	adb.CreateLiveSession(span.Context(), &models.LiveSession{
		ID: "abcdef",
	})

	adb.CreateLiveLesson(span.Context(), &models.LiveLesson{
		ID:        "foobar1",
		SessionID: "abcdef",
	})
	adb.CreateLiveLesson(span.Context(), &models.LiveLesson{
		ID:        "foobar2",
		SessionID: "uvwxyz",
	})
	adb.CreateLiveLesson(span.Context(), &models.LiveLesson{
		ID:        "foobar3",
		SessionID: "abcdef",
	})

	l, _ := adb.GetLiveLessonsForSession(span.Context(), "abcdef")
	equals(t, 2, len(l))

}
