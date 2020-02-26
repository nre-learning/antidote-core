package db

import (
	"math/rand"
	"time"

	models "github.com/nre-learning/antidote-core/db/models"
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
	ListImages() (map[string]models.Image, error)
	GetImage(string) (*models.Image, error)

	// Lessons
	InsertLessons([]*models.Lesson) error
	ListLessons() (map[string]models.Lesson, error)
	GetLesson(string) (*models.Lesson, error)

	// Collections
	InsertCollections([]*models.Collection) error
	ListCollections() (map[string]models.Collection, error)
	GetCollection(string) (*models.Collection, error)

	// Curriculum
	SetCurriculum(*models.Curriculum) error
	GetCurriculum() (*models.Curriculum, error)

	// LiveLessons
	CreateLiveLesson(*models.LiveLesson) error
	ListLiveLessons() (map[string]models.LiveLesson, error)
	GetLiveLesson(string) (*models.LiveLesson, error)
	UpdateLiveLesson(*models.LiveLesson) error
	DeleteLiveLesson(string) error

	// LiveSessions
	CreateLiveSession(*models.LiveSession) error
	ListLiveSessions() (map[string]models.LiveSession, error)
	GetLiveSession(string) (*models.LiveSession, error)
	UpdateLiveSession(*models.LiveSession) error
	DeleteLiveSession(string) error
}

// RandomID is a helper function designed to promote the unique creation of IDs for
// LiveLessons, LiveSesions, and other state resources that require such a unique identifier.
// No caller for this function should assume global uniqueness, but rather use this as a quick
// and easy way of generating something that is **probably** unique. Once generated,
// the caller should then check to ensure that ID is not already in use where it intends to use it,
// and in the unlikely event that it is, re-run this function until a unique value is determined.
// In this way, we can keep these IDs fairly small, which is necessary since we have to use them
// in forming names with constraints, like kubernetes objects.
func RandomID(length int) string {

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	seededRand := rand.New(
		rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
