package db

import (
	"math/rand"
	"time"

	models "github.com/nre-learning/antidote-core/db/models"
	ot "github.com/opentracing/opentracing-go"
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
	Preflight(ot.SpanContext) error
	Initialize(ot.SpanContext) error

	// Images
	InsertImages(ot.SpanContext, []*models.Image) error
	ListImages(ot.SpanContext) (map[string]models.Image, error)
	GetImage(ot.SpanContext, string) (models.Image, error)

	// Lessons
	InsertLessons(ot.SpanContext, []*models.Lesson) error
	ListLessons(ot.SpanContext) (map[string]models.Lesson, error)
	GetLesson(ot.SpanContext, string) (models.Lesson, error)

	// Collections
	InsertCollections(ot.SpanContext, []*models.Collection) error
	ListCollections(ot.SpanContext) (map[string]models.Collection, error)
	GetCollection(ot.SpanContext, string) (models.Collection, error)

	// Curriculum
	SetCurriculum(ot.SpanContext, *models.Curriculum) error
	GetCurriculum(ot.SpanContext) (models.Curriculum, error)

	// LiveLessons
	CreateLiveLesson(ot.SpanContext, *models.LiveLesson) error
	ListLiveLessons(ot.SpanContext) (map[string]models.LiveLesson, error)
	GetLiveLesson(ot.SpanContext, string) (models.LiveLesson, error)
	/*
		I started with a basic UpdateLiveLesson function, and then in the code, I'd first call GetLiveLesson,
		make some modifications, and then run UpdateLiveLesson. The problem is, if there are any changes to the
		livelesson between these two points in time, I'd overwrite those changes inadvertently.

		So, I decided to try with these specific update functions that are designed to update a specific field.
		This way you don't have to worry about the specific state, and you update only the field you intend.
		I thought about maybe changing the update function to take in a field name by string but that felt
		sinful in the face of strong typing, so I instead opted for the safe option and just created unique functions
		for each field that fits a use case.

		The first param is the livelesson ID, and the second is the appropriate value
	*/
	UpdateLiveLessonStage(ot.SpanContext, string, int32) error
	UpdateLiveLessonGuide(ot.SpanContext, string, string, string) error
	UpdateLiveLessonStatus(ot.SpanContext, string, models.LiveLessonStatus) error
	UpdateLiveLessonError(ot.SpanContext, string, bool) error
	UpdateLiveLessonEndpointIP(ot.SpanContext, string, string, string) error //ID, epName, IP
	UpdateLiveLessonTests(ot.SpanContext, string, int32, int32) error        //ID, healthyTests, totalTests
	DeleteLiveLesson(ot.SpanContext, string) error

	// LiveSessions
	CreateLiveSession(ot.SpanContext, *models.LiveSession) error
	ListLiveSessions(ot.SpanContext) (map[string]models.LiveSession, error)
	GetLiveSession(ot.SpanContext, string) (models.LiveSession, error)
	UpdateLiveSessionPersistence(ot.SpanContext, string, bool) error
	DeleteLiveSession(ot.SpanContext, string) error
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
