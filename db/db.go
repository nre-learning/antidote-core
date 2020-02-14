package db

import (
	models "github.com/nre-learning/syringe/db/models"
)

// DataManager enforces the set of functions required by the rest of the Antidote codebase for
// handling internal data, such as live state or loaded curriculum resource definitions.
//
// In general, resource types like collections or lessons only need three operations: Insert, List, Get
// They generally don't need to be deleted - the better option is to delete them from the source filesystem
// or repo, and then run a re-import, which first deletes everything and re-inserts them.
//
// State types like LiveLesson or LiveSession are different, and need full CRUD functionality.
type DataManager interface {

	// Housekeeping functions
	Preflight() error
	Initialize() error

	// Images
	InsertImages([]*models.Image) error
	ListImages() ([]*models.Image, error)
	GetImage(string) (*models.Image, error)

	// Lessons
	InsertLessons([]*models.Lesson) error
	ListLessons() ([]*models.Lesson, error)
	GetLesson(string) (*models.Lesson, error)

	// Collections
	InsertCollection([]*models.Collection) error
	ListCollections() ([]*models.Collection, error)
	GetCollection(string) (*models.Collection, error)

	// Curriculum
	SetCurriculum(*models.Curriculum) error
	GetCurriculum() (*models.Curriculum, error)

	// LiveLessons
	CreateLiveLesson(*models.LiveLesson) error
	ListLiveLessons() ([]*models.LiveLesson, error)
	GetLiveLesson(string) (*models.LiveLesson, error)
	UpdateLiveLesson(*models.LiveLesson) error
	DeleteLiveLesson(string) error

	// LiveSessions
	CreateLiveSessions(*models.LiveSession) error
	ListLiveSessions() ([]*models.LiveSession, error)
	GetLiveSession(string) (*models.LiveSession, error)
	UpdateLiveSession(*models.LiveSession) error
	DeleteLiveSession(string) error
}
