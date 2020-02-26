package db

import (
	"fmt"
	"sync"

	models "github.com/nre-learning/antidote-core/db/models"
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

var _ DataManager = &ADMInMem{}

// HOUSEKEEPING

// Preflight performs any necessary tasks to ensure the database is ready to be used.
// This includes things like version compatibilty checks, schema checks, the presence of the
// expected data, etc. Most useful for when Antidote processes start up.
//
// This function is left blank for the in-memory driver, as it's not needed.
func (a *ADMInMem) Preflight() error {
	return nil
}

// Initialize resets an Antidote datastore to its defaults. This is done by dropping any existing data
// or schema, and re-installing it from the embedded types. A very destructive operation - use with caution.
//
// This function is left blank for the in-memory driver, as it's not needed.
func (a *ADMInMem) Initialize() error {
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
func (a *ADMInMem) InsertLessons(lessons []*models.Lesson) error {
	a.lessonsMu.Lock()
	defer a.lessonsMu.Unlock()
	for i := range lessons {
		a.lessons[lessons[i].Slug] = lessons[i]
	}
	return nil
}

// ListLessons lists the Lessons currently available in the data store
func (a *ADMInMem) ListLessons() (map[string]models.Lesson, error) {
	lessons := map[string]models.Lesson{}
	for slug, lesson := range a.lessons {
		lessons[slug] = *lesson
	}
	return lessons, nil
}

// GetLesson retrieves a specific lesson from the data store
func (a *ADMInMem) GetLesson(slug string) (*models.Lesson, error) {
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
func (a *ADMInMem) InsertImages(images []*models.Image) error {
	a.imagesMu.Lock()
	defer a.imagesMu.Unlock()
	for i := range images {
		a.images[images[i].Slug] = images[i]
	}
	return nil
}

// ListImages lists the Images currently available in the data store
func (a *ADMInMem) ListImages() (map[string]models.Image, error) {
	images := map[string]models.Image{}
	for slug, image := range a.images {
		images[slug] = *image
	}
	return images, nil
}

// GetImage retrieves a specific Image from the data store
func (a *ADMInMem) GetImage(slug string) (*models.Image, error) {
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
func (a *ADMInMem) InsertCollections(collections []*models.Collection) error {
	a.collectionsMu.Lock()
	defer a.collectionsMu.Unlock()
	for i := range collections {
		a.collections[collections[i].Slug] = collections[i]
	}
	return nil
}

// ListCollections lists the Collections currently available in the data store
func (a *ADMInMem) ListCollections() (map[string]models.Collection, error) {
	collections := map[string]models.Collection{}
	for slug, collection := range a.collections {
		collections[slug] = *collection
	}
	return collections, nil
}

// GetCollection retrieves a specific Collection from the data store
func (a *ADMInMem) GetCollection(slug string) (*models.Collection, error) {
	if collection, ok := a.collections[slug]; ok {
		return collection, nil
	}
	return nil, fmt.Errorf("Unable to find collection %s", slug)
}

// CURRICULUM

// SetCurriculum updates the curriculum details in the datastore
func (a *ADMInMem) SetCurriculum(curriculum *models.Curriculum) error {
	a.curriculumMu.Lock()
	defer a.collectionsMu.Unlock()
	a.curriculum = curriculum
	return nil
}

// GetCurriculum retrieves a specific Curriculum from the data store
func (a *ADMInMem) GetCurriculum() (*models.Curriculum, error) {
	return a.curriculum, nil
}

// LIVELESSONS

// CreateLiveLesson creates a new instance of a LiveLesson to the in-memory data store
func (a *ADMInMem) CreateLiveLesson(ll *models.LiveLesson) error {
	if _, ok := a.liveLessons[ll.ID]; ok {
		return fmt.Errorf("Livelesson %s already exists", ll.ID)
	}
	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[ll.ID] = ll

	log.Infof("Created livelesson %s", ll.ID)
	return nil
}

// ListLiveLessons lists all LiveLessons currently tracked in memory
func (a *ADMInMem) ListLiveLessons() (map[string]models.LiveLesson, error) {
	liveLessons := map[string]models.LiveLesson{}
	for id, ll := range a.liveLessons {
		liveLessons[id] = *ll
	}

	log.Info("Retrieving all livelessons")
	return liveLessons, nil

}

// GetLiveLesson retrieves a specific LiveLesson from the in-memory store via ID
func (a *ADMInMem) GetLiveLesson(id string) (*models.LiveLesson, error) {
	if ll, ok := a.liveLessons[id]; ok {
		return ll, nil
	}
	return nil, fmt.Errorf("Unable to find liveLesson %s", id)
}

// UpdateLiveLesson updates an existing LiveLesson in-place within the in-memory data store, by ID
func (a *ADMInMem) UpdateLiveLesson(ll *models.LiveLesson) error {
	if _, ok := a.liveLessons[ll.ID]; !ok {
		return fmt.Errorf("Livelesson %s doesn't exist; cannot update", ll.ID)
	}
	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	a.liveLessons[ll.ID] = ll
	return nil
}

// DeleteLiveLesson deletes an existing LiveLesson from the in-memory data store by ID
func (a *ADMInMem) DeleteLiveLesson(id string) error {
	a.liveLessonsMu.Lock()
	defer a.liveLessonsMu.Unlock()
	delete(a.liveLessons, id)
	return nil
}

// LIVESESSIONS

// CreateLiveSession creates a new instance of a LiveSession to the in-memory data store
func (a *ADMInMem) CreateLiveSession(ls *models.LiveSession) error {
	if _, ok := a.liveSessions[ls.ID]; ok {
		return fmt.Errorf("LiveSession %s already exists", ls.ID)
	}
	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	a.liveSessions[ls.ID] = ls
	return nil
}

// ListLiveSessions lists all LiveSessions currently tracked in memory
func (a *ADMInMem) ListLiveSessions() (map[string]models.LiveSession, error) {
	liveSessions := map[string]models.LiveSession{}
	for id, ls := range a.liveSessions {
		liveSessions[id] = *ls
	}
	return liveSessions, nil
}

// GetLiveSession retrieves a specific LiveSession from the in-memory store via ID
func (a *ADMInMem) GetLiveSession(id string) (*models.LiveSession, error) {
	if ls, ok := a.liveSessions[id]; ok {
		return ls, nil
	}
	return nil, fmt.Errorf("Unable to find liveSession %s", id)
}

// UpdateLiveSession updates an existing LiveSession in-place within the in-memory data store, by ID
func (a *ADMInMem) UpdateLiveSession(ls *models.LiveSession) error {
	if _, ok := a.liveSessions[ls.ID]; !ok {
		return fmt.Errorf("LiveSession %s doesn't exist; cannot update", ls.ID)
	}
	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	a.liveSessions[ls.ID] = ls
	return nil
}

// DeleteLiveSession deletes an existing LiveSession from the in-memory data store by ID
func (a *ADMInMem) DeleteLiveSession(id string) error {
	a.liveSessionsMu.Lock()
	defer a.liveSessionsMu.Unlock()
	delete(a.liveSessions, id)
	return nil
}
