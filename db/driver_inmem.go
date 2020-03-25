package db

import (
	"fmt"
	"sync"

	models "github.com/nre-learning/antidote-core/db/models"
	"github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
)

// NewADMInMem produces an initialized instance of ADMInMem ready to be used
func NewADMInMem() DataManager {

	return &ADMInMem{
		// AntidoteVersion string
		lessons:        map[string]*models.Lesson{},
		lessonsMu:      &sync.Mutex{},
		collections:    map[string]*models.Collection{},
		collectionsMu:  &sync.Mutex{},
		images:         map[string]*models.Image{},
		imagesMu:       &sync.Mutex{},
		curriculum:     &models.Curriculum{},
		curriculumMu:   &sync.Mutex{},
		liveLessons:    map[string]*models.LiveLesson{},
		liveLessonsMu:  &sync.Mutex{},
		liveSessions:   map[string]*models.LiveSession{},
		liveSessionsMu: &sync.Mutex{},
	}

}

// ADMInMem is an implementation of DataManager which uses in-memory
// constructs as a backing data store
type ADMInMem struct {
	AntidoteVersion string

	// All fields should be unexported; since these are managed in memory, they should only be accessible through
	// exported functions in this driver that allow this to be done safely
	lessons       map[string]*models.Lesson
	lessonsMu     *sync.Mutex
	collections   map[string]*models.Collection
	collectionsMu *sync.Mutex
	images        map[string]*models.Image
	imagesMu      *sync.Mutex
	curriculum    *models.Curriculum
	curriculumMu  *sync.Mutex

	liveLessons    map[string]*models.LiveLesson
	liveLessonsMu  *sync.Mutex
	liveSessions   map[string]*models.LiveSession
	liveSessionsMu *sync.Mutex
}

// TODO(mierdin): Add span event logs

var _ DataManager = &ADMInMem{}

// HOUSEKEEPING

// Preflight performs any necessary tasks to ensure the database is ready to be used.
// This includes things like version compatibilty checks, schema checks, the presence of the
// expected data, etc. Most useful for when Antidote processes start up.
//
// This function is left blank for the in-memory driver, as it's not needed.
func (a *ADMInMem) Preflight(sc opentracing.SpanContext) error {
	return nil
}

// Initialize resets an Antidote datastore to its defaults. This is done by dropping any existing data
// or schema, and re-installing it from the embedded types. A very destructive operation - use with caution.
//
// This function is left blank for the in-memory driver, as it's not needed.
func (a *ADMInMem) Initialize(sc opentracing.SpanContext) error {
	return nil
}

// LESSONS

// InsertLessons takes a slice of Lessons, and creates entries for each in the in-memory
// store.
//
// NOTE that this and other insert operations silently overwrite any existing entities.
// This is okay for this driver, but we may want to revisit this later for other drivers
// especially. What's the appropriate behavior when we're trying to insert an item
// that already exists?
func (a *ADMInMem) InsertLessons(sc opentracing.SpanContext, lessons []*models.Lesson) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_lesson_insert",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.lessonsMu.Lock()
	defer a.lessonsMu.Unlock()
	for i := range lessons {
		a.lessons[lessons[i].Slug] = lessons[i]
	}
	return nil
}

// ListLessons lists the Lessons currently available in the data store
func (a *ADMInMem) ListLessons(sc opentracing.SpanContext) (map[string]models.Lesson, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_lesson_list",
		opentracing.ChildOf(sc))
	defer span.Finish()

	lessons := map[string]models.Lesson{}
	for slug, lesson := range a.lessons {
		lessons[slug] = *lesson
	}
	return lessons, nil
}

// GetLesson retrieves a specific lesson from the data store
func (a *ADMInMem) GetLesson(sc opentracing.SpanContext, slug string) (*models.Lesson, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_lesson_get",
		opentracing.ChildOf(sc))
	defer span.Finish()

	if lesson, ok := a.lessons[slug]; ok {
		return lesson, nil
	}
	return nil, fmt.Errorf("Unable to find lesson %s", slug)
}

// IMAGES

// InsertImages takes a slice of Images, and creates entries for each in the in-memory
// store.
//
// NOTE that this and other insert operations silently overwrite any existing entities.
// This is okay for this driver, but we may want to revisit this later for other drivers
// especially. What's the appropriate behavior when we're trying to insert an item
// that already exists?
func (a *ADMInMem) InsertImages(sc opentracing.SpanContext, images []*models.Image) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_image_insert",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.imagesMu.Lock()
	defer a.imagesMu.Unlock()
	for i := range images {
		a.images[images[i].Slug] = images[i]
	}
	return nil
}

// ListImages lists the Images currently available in the data store
func (a *ADMInMem) ListImages(sc opentracing.SpanContext) (map[string]models.Image, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_image_list",
		opentracing.ChildOf(sc))
	defer span.Finish()

	images := map[string]models.Image{}
	for slug, image := range a.images {
		images[slug] = *image
	}
	return images, nil
}

// GetImage retrieves a specific Image from the data store
func (a *ADMInMem) GetImage(sc opentracing.SpanContext, slug string) (*models.Image, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_image_get",
		opentracing.ChildOf(sc))
	defer span.Finish()

	if image, ok := a.images[slug]; ok {
		return image, nil
	}
	return nil, fmt.Errorf("Unable to find image %s", slug)
}

// COLLECTIONS

// InsertCollections takes a slice of Images, and creates entries for each in the in-memory
// store.
//
// NOTE that this and other insert operations silently overwrite any existing entities.
// This is okay for this driver, but we may want to revisit this later for other drivers
// especially. What's the appropriate behavior when we're trying to insert an item
// that already exists?
func (a *ADMInMem) InsertCollections(sc opentracing.SpanContext, collections []*models.Collection) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_collection_insert",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.collectionsMu.Lock()
	defer a.collectionsMu.Unlock()
	for i := range collections {
		a.collections[collections[i].Slug] = collections[i]
	}
	return nil
}

// ListCollections lists the Collections currently available in the data store
func (a *ADMInMem) ListCollections(sc opentracing.SpanContext) (map[string]models.Collection, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_collection_list",
		opentracing.ChildOf(sc))
	defer span.Finish()

	collections := map[string]models.Collection{}
	for slug, collection := range a.collections {
		collections[slug] = *collection
	}
	return collections, nil
}

// GetCollection retrieves a specific Collection from the data store
func (a *ADMInMem) GetCollection(sc opentracing.SpanContext, slug string) (*models.Collection, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_collection_get",
		opentracing.ChildOf(sc))
	defer span.Finish()

	if collection, ok := a.collections[slug]; ok {
		return collection, nil
	}
	return nil, fmt.Errorf("Unable to find collection %s", slug)
}

// CURRICULUM

// SetCurriculum updates the curriculum details in the datastore
func (a *ADMInMem) SetCurriculum(sc opentracing.SpanContext, curriculum *models.Curriculum) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_curriculum_set",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.curriculumMu.Lock()
	defer a.collectionsMu.Unlock()
	a.curriculum = curriculum
	return nil
}

// GetCurriculum retrieves a specific Curriculum from the data store
func (a *ADMInMem) GetCurriculum(sc opentracing.SpanContext) (*models.Curriculum, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_curriculum_get",
		opentracing.ChildOf(sc))
	defer span.Finish()

	return a.curriculum, nil
}

// LIVELESSONS

// CreateLiveLesson creates a new instance of a LiveLesson to the in-memory data store
func (a *ADMInMem) CreateLiveLesson(sc opentracing.SpanContext, ll *models.LiveLesson) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_create",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	if _, ok := a.liveLessons[ll.ID]; ok {
		return fmt.Errorf("Livelesson %s already exists", ll.ID)
	}
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[ll.ID] = ll

	log.Infof("Created livelesson %s", ll.ID)
	return nil
}

// ListLiveLessons lists all LiveLessons currently tracked in memory
func (a *ADMInMem) ListLiveLessons(sc opentracing.SpanContext) (map[string]models.LiveLesson, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_list",
		opentracing.ChildOf(sc))
	defer span.Finish()

	liveLessons := map[string]models.LiveLesson{}
	for id, ll := range a.liveLessons {
		liveLessons[id] = *ll
	}

	log.Info("Retrieving all livelessons")
	return liveLessons, nil

}

// GetLiveLesson retrieves a specific LiveLesson from the in-memory store via ID
func (a *ADMInMem) GetLiveLesson(sc opentracing.SpanContext, id string) (*models.LiveLesson, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_get",
		opentracing.ChildOf(sc))
	defer span.Finish()

	if ll, ok := a.liveLessons[id]; ok {
		return ll, nil
	}
	return nil, fmt.Errorf("Unable to find liveLesson %s", id)
}

// UpdateLiveLessonStage updates a livelesson's LessonStage property
func (a *ADMInMem) UpdateLiveLessonStage(sc opentracing.SpanContext, llID string, stage int32) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_update_stage",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	if _, ok := a.liveLessons[llID]; !ok {
		return fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
	}
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[llID].CurrentStage = stage
	return nil
}

// UpdateLiveLessonGuide updates a LiveLesson's guide properties
func (a *ADMInMem) UpdateLiveLessonGuide(sc opentracing.SpanContext, llID, guideType, guideContents string) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_update_guide",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	if _, ok := a.liveLessons[llID]; !ok {
		return fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
	}
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[llID].GuideContents = guideContents
	a.liveLessons[llID].GuideType = guideType
	return nil
}

// UpdateLiveLessonStatus updates a livelesson's Status property
func (a *ADMInMem) UpdateLiveLessonStatus(sc opentracing.SpanContext, llID string, status models.LiveLessonStatus) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_update_status",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	if _, ok := a.liveLessons[llID]; !ok {
		return fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
	}
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[llID].Status = status
	return nil
}

// UpdateLiveLessonError updates a livelesson's Error property
func (a *ADMInMem) UpdateLiveLessonError(sc opentracing.SpanContext, llID string, err bool) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_update_error",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	if _, ok := a.liveLessons[llID]; !ok {
		return fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
	}
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[llID].Error = err
	return nil
}

// UpdateLiveLessonEndpointIP updates a livelesson's Host property
func (a *ADMInMem) UpdateLiveLessonEndpointIP(sc opentracing.SpanContext, llID, epName, IP string) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_update_endpointip",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	if _, ok := a.liveLessons[llID]; !ok {
		return fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
	}
	defer a.liveLessonsMu.Unlock()
	for name := range a.liveLessons[llID].LiveEndpoints {
		if name == epName {
			a.liveLessons[llID].LiveEndpoints[name].Host = IP
			break
		}
	}
	return nil
}

// DeleteLiveLesson deletes an existing LiveLesson from the in-memory data store by ID
func (a *ADMInMem) DeleteLiveLesson(sc opentracing.SpanContext, id string) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livelesson_delete",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	delete(a.liveLessons, id)
	return nil
}

// LIVESESSIONS

// CreateLiveSession creates a new instance of a LiveSession to the in-memory data store
func (a *ADMInMem) CreateLiveSession(sc opentracing.SpanContext, ls *models.LiveSession) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livesession_create",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveSessionsMu.Lock()
	if _, ok := a.liveSessions[ls.ID]; ok {
		return fmt.Errorf("LiveSession %s already exists", ls.ID)
	}
	defer a.liveSessionsMu.Unlock()
	a.liveSessions[ls.ID] = ls
	return nil
}

// ListLiveSessions lists all LiveSessions currently tracked in memory
func (a *ADMInMem) ListLiveSessions(sc opentracing.SpanContext) (map[string]models.LiveSession, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livesession_list",
		opentracing.ChildOf(sc))
	defer span.Finish()

	liveSessions := map[string]models.LiveSession{}
	for id, ls := range a.liveSessions {
		liveSessions[id] = *ls
	}
	return liveSessions, nil
}

// GetLiveSession retrieves a specific LiveSession from the in-memory store via ID
func (a *ADMInMem) GetLiveSession(sc opentracing.SpanContext, id string) (*models.LiveSession, error) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livesession_get",
		opentracing.ChildOf(sc))
	defer span.Finish()

	if ls, ok := a.liveSessions[id]; ok {
		return ls, nil
	}
	return nil, fmt.Errorf("Unable to find liveSession %s", id)
}

// UpdateLiveSessionPersistence updates a livesession's persistent property
func (a *ADMInMem) UpdateLiveSessionPersistence(sc opentracing.SpanContext, lsID string, persistent bool) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livesession_update_persistence",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveSessionsMu.Lock()
	if _, ok := a.liveSessions[lsID]; !ok {
		return fmt.Errorf("Livesesson %s doesn't exist; cannot update", lsID)
	}
	defer a.liveSessionsMu.Unlock()
	a.liveSessions[lsID].Persistent = persistent
	return nil
}

// DeleteLiveSession deletes an existing LiveSession from the in-memory data store by ID
func (a *ADMInMem) DeleteLiveSession(sc opentracing.SpanContext, id string) error {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"db_livesession_delete",
		opentracing.ChildOf(sc))
	defer span.Finish()

	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	delete(a.liveSessions, id)
	return nil
}
