package db

import (
	"fmt"
	"sync"

	models "github.com/nre-learning/antidote-core/db/models"
	ot "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	log "github.com/opentracing/opentracing-go/log"
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

var _ DataManager = &ADMInMem{}

// HOUSEKEEPING

// Preflight performs any necessary tasks to ensure the database is ready to be used.
// This includes things like version compatibilty checks, schema checks, the presence of the
// expected data, etc. Most useful for when Antidote processes start up.
//
// This function is left blank for the in-memory driver, as it's not needed.
func (a *ADMInMem) Preflight(sc ot.SpanContext) error {
	return nil
}

// Initialize resets an Antidote datastore to its defaults. This is done by dropping any existing data
// or schema, and re-installing it from the embedded types. A very destructive operation - use with caution.
//
// This function is left blank for the in-memory driver, as it's not needed.
func (a *ADMInMem) Initialize(sc ot.SpanContext) error {
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
func (a *ADMInMem) InsertLessons(sc ot.SpanContext, lessons []*models.Lesson) error {
	span := ot.StartSpan("db_lesson_insert", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("insertLength", len(lessons))

	a.lessonsMu.Lock()
	defer a.lessonsMu.Unlock()
	for i := range lessons {
		a.lessons[lessons[i].Slug] = lessons[i]
	}

	return nil
}

// ListLessons lists the Lessons currently available in the data store
func (a *ADMInMem) ListLessons(sc ot.SpanContext) (map[string]models.Lesson, error) {
	span := ot.StartSpan("db_lesson_list", ot.ChildOf(sc))
	defer span.Finish()
	a.lessonsMu.Lock()
	defer a.lessonsMu.Unlock()

	lessons := map[string]models.Lesson{}
	for slug, lesson := range a.lessons {
		lessons[slug] = *lesson
	}
	span.LogFields(log.Int("numLessons", len(lessons)))
	return lessons, nil
}

// GetLesson retrieves a specific lesson from the data store
func (a *ADMInMem) GetLesson(sc ot.SpanContext, slug string) (models.Lesson, error) {
	span := ot.StartSpan("db_lesson_get", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("lessonSlug", slug)

	if lesson, ok := a.lessons[slug]; ok {
		return *lesson, nil
	}
	err := fmt.Errorf("Unable to find lesson %s", slug)
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return models.Lesson{}, err
}

// IMAGES

// InsertImages takes a slice of Images, and creates entries for each in the in-memory
// store.
//
// NOTE that this and other insert operations silently overwrite any existing entities.
// This is okay for this driver, but we may want to revisit this later for other drivers
// especially. What's the appropriate behavior when we're trying to insert an item
// that already exists?
func (a *ADMInMem) InsertImages(sc ot.SpanContext, images []*models.Image) error {
	span := ot.StartSpan("db_image_insert", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("insertLength", len(images))

	a.imagesMu.Lock()
	defer a.imagesMu.Unlock()
	for i := range images {
		a.images[images[i].Slug] = images[i]
	}
	return nil
}

// ListImages lists the Images currently available in the data store
func (a *ADMInMem) ListImages(sc ot.SpanContext) (map[string]models.Image, error) {
	span := ot.StartSpan("db_image_list", ot.ChildOf(sc))
	defer span.Finish()
	a.imagesMu.Lock()
	defer a.imagesMu.Unlock()

	images := map[string]models.Image{}
	for slug, image := range a.images {
		images[slug] = *image
	}
	span.LogFields(log.Int("numImages", len(images)))
	return images, nil
}

// GetImage retrieves a specific Image from the data store
func (a *ADMInMem) GetImage(sc ot.SpanContext, slug string) (models.Image, error) {
	span := ot.StartSpan("db_image_get", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("imageSlug", slug)

	if image, ok := a.images[slug]; ok {
		return *image, nil
	}
	err := fmt.Errorf("Unable to find image %s", slug)
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return models.Image{}, err
}

// COLLECTIONS

// InsertCollections takes a slice of Images, and creates entries for each in the in-memory
// store.
//
// NOTE that this and other insert operations silently overwrite any existing entities.
// This is okay for this driver, but we may want to revisit this later for other drivers
// especially. What's the appropriate behavior when we're trying to insert an item
// that already exists?
func (a *ADMInMem) InsertCollections(sc ot.SpanContext, collections []*models.Collection) error {
	span := ot.StartSpan("db_collection_insert", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("insertLength", len(collections))

	a.collectionsMu.Lock()
	defer a.collectionsMu.Unlock()
	for i := range collections {
		a.collections[collections[i].Slug] = collections[i]
	}
	return nil
}

// ListCollections lists the Collections currently available in the data store
func (a *ADMInMem) ListCollections(sc ot.SpanContext) (map[string]models.Collection, error) {
	span := ot.StartSpan("db_collection_list", ot.ChildOf(sc))
	defer span.Finish()
	a.collectionsMu.Lock()
	defer a.collectionsMu.Unlock()

	collections := map[string]models.Collection{}
	for slug, collection := range a.collections {
		collections[slug] = *collection
	}
	span.LogFields(log.Int("numCollections", len(collections)))
	return collections, nil
}

// GetCollection retrieves a specific Collection from the data store
func (a *ADMInMem) GetCollection(sc ot.SpanContext, slug string) (models.Collection, error) {
	span := ot.StartSpan("db_collection_get", ot.ChildOf(sc))
	defer span.Finish()

	if collection, ok := a.collections[slug]; ok {
		return *collection, nil
	}
	err := fmt.Errorf("Unable to find collection %s", slug)
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return models.Collection{}, err
}

// CURRICULUM

// SetCurriculum updates the curriculum details in the datastore
func (a *ADMInMem) SetCurriculum(sc ot.SpanContext, curriculum *models.Curriculum) error {
	span := ot.StartSpan("db_curriculum_set", ot.ChildOf(sc))
	defer span.Finish()

	// NOTE that I'm only doing this because the Curriculum model is pretty small/simple.
	// I would recommend doing this convervatively, to not bloat our spans. For nearly all functions
	// in this package, I'm only logging partial or aggregate information about input/output
	span.LogFields(log.Object("curriculum", curriculum))

	a.curriculumMu.Lock()
	defer a.curriculumMu.Unlock()
	a.curriculum = curriculum
	return nil
}

// GetCurriculum retrieves a specific Curriculum from the data store
func (a *ADMInMem) GetCurriculum(sc ot.SpanContext) (models.Curriculum, error) {
	span := ot.StartSpan("db_curriculum_get", ot.ChildOf(sc))
	defer span.Finish()

	return *a.curriculum, nil
}

// LIVELESSONS

// CreateLiveLesson creates a new instance of a LiveLesson to the in-memory data store
func (a *ADMInMem) CreateLiveLesson(sc ot.SpanContext, ll *models.LiveLesson) error {
	span := ot.StartSpan("db_livelesson_create", ot.ChildOf(sc))
	defer span.Finish()

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[ll.ID]; ok {
		err := fmt.Errorf("Livelesson %s already exists", ll.ID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveLessons[ll.ID] = ll

	span.LogFields(log.String("createdLLID", ll.ID))
	return nil
}

// ListLiveLessons lists all LiveLessons currently tracked in memory
func (a *ADMInMem) ListLiveLessons(sc ot.SpanContext) (map[string]models.LiveLesson, error) {
	span := ot.StartSpan("db_livelesson_list", ot.ChildOf(sc))
	defer span.Finish()
	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()

	liveLessons := map[string]models.LiveLesson{}
	for id, ll := range a.liveLessons {
		liveLessons[id] = *ll
	}
	span.LogFields(log.Int("numLiveLessons", len(liveLessons)))
	return liveLessons, nil
}

// GetLiveLesson retrieves a specific LiveLesson from the in-memory store via ID
func (a *ADMInMem) GetLiveLesson(sc ot.SpanContext, id string) (models.LiveLesson, error) {
	span := ot.StartSpan("db_livelesson_get", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", id)

	if ll, ok := a.liveLessons[id]; ok {
		return *ll, nil
	}
	err := fmt.Errorf("Unable to find liveLesson %s", id)
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return models.LiveLesson{}, err
}

// UpdateLiveLessonStage updates a livelesson's LessonStage property
func (a *ADMInMem) UpdateLiveLessonStage(sc ot.SpanContext, llID string, stage int32) error {
	span := ot.StartSpan("db_livelesson_update_stage", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", llID)
	span.SetTag("stage", stage)

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[llID]; !ok {
		err := fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveLessons[llID].CurrentStage = stage
	return nil
}

// UpdateLiveLessonGuide updates a LiveLesson's guide properties
func (a *ADMInMem) UpdateLiveLessonGuide(sc ot.SpanContext, llID, guideType, guideContents string) error {
	span := ot.StartSpan("db_livelesson_update_guide", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", llID)
	span.SetTag("guideType", guideType)
	// do NOT include guideContents in a span, that is likely to be huge

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[llID]; !ok {
		err := fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveLessons[llID].GuideContents = guideContents
	a.liveLessons[llID].GuideType = guideType
	return nil
}

// UpdateLiveLessonStatus updates a livelesson's Status property
func (a *ADMInMem) UpdateLiveLessonStatus(sc ot.SpanContext, llID string, status models.LiveLessonStatus) error {
	span := ot.StartSpan("db_livelesson_update_status", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", llID)
	span.SetTag("status", status)

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[llID]; !ok {
		err := fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveLessons[llID].Status = status
	return nil
}

// UpdateLiveLessonError updates a livelesson's Error property
func (a *ADMInMem) UpdateLiveLessonError(sc ot.SpanContext, llID string, llErr bool) error {
	span := ot.StartSpan("db_livelesson_update_error", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", llID)
	span.SetTag("liveLessonError", llErr)

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[llID]; !ok {
		err := fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveLessons[llID].Error = llErr
	return nil
}

// UpdateLiveLessonEndpointIP updates a livelesson's Host property
func (a *ADMInMem) UpdateLiveLessonEndpointIP(sc ot.SpanContext, llID, epName, IP string) error {
	span := ot.StartSpan("db_livelesson_update_endpointip", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", llID)
	span.SetTag("liveLessonEPIP", IP)

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[llID]; !ok {
		err := fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	for name := range a.liveLessons[llID].LiveEndpoints {
		if name == epName {
			a.liveLessons[llID].LiveEndpoints[name].Host = IP
			break
		}
	}
	return nil
}

// UpdateLiveLessonTests updates the HealthyTests and TotalTests properties
func (a *ADMInMem) UpdateLiveLessonTests(sc ot.SpanContext, llID string, healthy, total int32) error {
	span := ot.StartSpan("db_livelesson_update_tests", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", llID)
	span.SetTag("healthy", healthy)
	span.SetTag("total", total)

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	if _, ok := a.liveLessons[llID]; !ok {
		err := fmt.Errorf("Livelesson %s doesn't exist; cannot update", llID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}

	a.liveLessons[llID].HealthyTests = healthy
	a.liveLessons[llID].TotalTests = total
	return nil
}

// DeleteLiveLesson deletes an existing LiveLesson from the in-memory data store by ID
func (a *ADMInMem) DeleteLiveLesson(sc ot.SpanContext, id string) error {
	span := ot.StartSpan("db_livelesson_delete", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("llID", id)

	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	delete(a.liveLessons, id)
	return nil
}

// LIVESESSIONS

// CreateLiveSession creates a new instance of a LiveSession to the in-memory data store
func (a *ADMInMem) CreateLiveSession(sc ot.SpanContext, ls *models.LiveSession) error {
	span := ot.StartSpan("db_livesession_create", ot.ChildOf(sc))
	defer span.Finish()

	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	if _, ok := a.liveSessions[ls.ID]; ok {
		err := fmt.Errorf("LiveSession %s already exists", ls.ID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveSessions[ls.ID] = ls

	span.LogFields(log.String("createdLSID", ls.ID))
	return nil
}

// ListLiveSessions lists all LiveSessions currently tracked in memory
func (a *ADMInMem) ListLiveSessions(sc ot.SpanContext) (map[string]models.LiveSession, error) {
	span := ot.StartSpan("db_livesession_list", ot.ChildOf(sc))
	defer span.Finish()
	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()

	liveSessions := map[string]models.LiveSession{}
	for id, ls := range a.liveSessions {
		liveSessions[id] = *ls
	}
	span.LogFields(log.Int("numLiveSessions", len(liveSessions)))
	return liveSessions, nil
}

// GetLiveSession retrieves a specific LiveSession from the in-memory store via ID
func (a *ADMInMem) GetLiveSession(sc ot.SpanContext, id string) (models.LiveSession, error) {
	span := ot.StartSpan("db_livesession_get", ot.ChildOf(sc))
	defer span.Finish()

	if ls, ok := a.liveSessions[id]; ok {
		return *ls, nil
	}
	err := fmt.Errorf("Unable to find liveSession %s", id)
	span.LogFields(log.Error(err))
	ext.Error.Set(span, true)
	return models.LiveSession{}, err
}

// UpdateLiveSessionPersistence updates a livesession's persistent property
func (a *ADMInMem) UpdateLiveSessionPersistence(sc ot.SpanContext, lsID string, persistent bool) error {
	span := ot.StartSpan("db_livesession_update_persistence", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("lsID", lsID)

	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	if _, ok := a.liveSessions[lsID]; !ok {
		err := fmt.Errorf("Livesesson %s doesn't exist; cannot update", lsID)
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return err
	}
	a.liveSessions[lsID].Persistent = persistent
	return nil
}

// DeleteLiveSession deletes an existing LiveSession from the in-memory data store by ID
func (a *ADMInMem) DeleteLiveSession(sc ot.SpanContext, id string) error {
	span := ot.StartSpan("db_livesession_delete", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("lsID", id)

	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	delete(a.liveSessions, id)
	return nil
}

// GetLiveLessonsForSession is a helper function to make it easier to look up all livelessons for a given session ID
func (a *ADMInMem) GetLiveLessonsForSession(sc ot.SpanContext, lsID string) ([]string, error) {
	span := ot.StartSpan("kubernetes_getlivelessonsforsession", ot.ChildOf(sc))
	defer span.Finish()
	span.SetTag("lsID", lsID)

	llList, err := a.ListLiveLessons(span.Context())
	if err != nil {
		span.LogFields(log.Error(err))
		ext.Error.Set(span, true)
		return nil, err
	}

	retLLIDs := []string{}

	for _, ll := range llList {
		if ll.SessionID == lsID {
			retLLIDs = append(retLLIDs, ll.ID)
		}
	}

	span.LogFields(
		log.Object("llIDs", retLLIDs),
		log.Int("llCount", len(retLLIDs)),
	)

	return retLLIDs, nil
}
