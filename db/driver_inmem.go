package db

import (
	"fmt"
	"sync"

	models "github.com/nre-learning/syringe/db/models"
)

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

	// Keep these unexported - since these are managed in memory, they should only be accessible through
	// exported functions in this package that allow this to be done safely
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

func (a *ADMInMem) Preflight() error {
	// Not needed for in-memory
	return nil
}

func (a *ADMInMem) Initialize() error {
	// Not needed for in-memory
	return nil
}

// LESSONS

func (a *ADMInMem) InsertLessons(lessons []*models.Lesson) error {
	a.lessonsMu.Lock()
	defer a.lessonsMu.Unlock()
	for i := range lessons {
		a.lessons[lessons[i].Slug] = lessons[i]
	}
	return nil
}

func (a *ADMInMem) ListLessons() ([]*models.Lesson, error) {
	lessons := []*models.Lesson{}
	for _, lesson := range a.lessons {
		lessons = append(lessons, lesson)
	}
	return lessons, nil
}

func (a *ADMInMem) GetLesson(slug string) (*models.Lesson, error) {
	if lesson, ok := a.lessons[slug]; ok {
		return lesson, nil
	}
	return nil, fmt.Errorf("Unable to find lesson %s", slug)
}

// IMAGES

func (a *ADMInMem) InsertImages(images []*models.Image) error {
	a.imagesMu.Lock()
	defer a.lessonsMu.Unlock()
	for i := range images {
		a.images[images[i].Slug] = images[i]
	}
	return nil
}

func (a *ADMInMem) ListImages() ([]*models.Image, error) {
	images := []*models.Image{}
	for _, image := range a.images {
		images = append(images, image)
	}
	return images, nil
}

func (a *ADMInMem) GetImage(slug string) (*models.Image, error) {
	if image, ok := a.images[slug]; ok {
		return image, nil
	}
	return nil, fmt.Errorf("Unable to find image %s", slug)
}

// COLLECTIONS

func (a *ADMInMem) InsertCollection(collections []*models.Collection) error {
	a.collectionsMu.Lock()
	defer a.lessonsMu.Unlock()
	for i := range collections {
		a.collections[collections[i].Slug] = collections[i]
	}
	return nil
}

func (a *ADMInMem) ListCollections() ([]*models.Collection, error) {
	collections := []*models.Collection{}
	for _, collection := range a.collections {
		collections = append(collections, collection)
	}
	return collections, nil
}

func (a *ADMInMem) GetCollection(slug string) (*models.Collection, error) {
	if collection, ok := a.collections[slug]; ok {
		return collection, nil
	}
	return nil, fmt.Errorf("Unable to find collection %s", slug)
}

// CURRICULUM

func (a *ADMInMem) SetCurriculum(curriculum *models.Curriculum) error {
	return nil
}

func (a *ADMInMem) GetCurriculum() (*models.Curriculum, error) {
	return &models.Curriculum{}, nil
}

// LIVELESSONS

func (a *ADMInMem) CreateLiveLesson(ll *models.LiveLesson) error {
	return nil
}

func (a *ADMInMem) ListLiveLessons() ([]*models.LiveLesson, error) {
	liveLessons := []*models.LiveLesson{}
	for _, liveLesson := range a.liveLessons {
		liveLessons = append(liveLessons, liveLesson)
	}
	return liveLessons, nil
}

func (a *ADMInMem) GetLiveLesson(id string) (*models.LiveLesson, error) {
	return &models.LiveLesson{}, nil
}

func (a *ADMInMem) UpdateLiveLesson(ll *models.LiveLesson) error {
	return nil
}

func (a *ADMInMem) DeleteLiveLesson(id string) error {
	return nil
}

// LIVESESSIONS

func (a *ADMInMem) CreateLiveSessions(ls *models.LiveSession) error {
	return nil
}

func (a *ADMInMem) ListLiveSessions() ([]*models.LiveSession, error) {
	liveSessions := []*models.LiveSession{}
	for _, liveSession := range a.liveSessions {
		liveSessions = append(liveSessions, liveSession)
	}
	return liveSessions, nil
}

func (a *ADMInMem) GetLiveSession(id string) (*models.LiveSession, error) {
	return &models.LiveSession{}, nil
}

func (a *ADMInMem) UpdateLiveSession(ls *models.LiveSession) error {
	return nil
}

func (a *ADMInMem) DeleteLiveSession(id string) error {
	return nil
}
