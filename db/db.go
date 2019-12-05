package db

import (
	pg "github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"

	models "github.com/nre-learning/syringe/db/models"
)

// Databaser defines all functions for the db layer
// Use this to provide a mock layer for tests
type Databaser interface {

	// Misc
	Preflight() error
	Initialize() error

	// Lessons
	CreateLesson(*models.Lesson) error
	ListLessons() ([]*models.Lesson, error)
	GetLesson(int) (*models.Lesson, error)
	UpdateLesson(*models.Lesson) error
	DeleteLesson(int) error

	// Collections
	CreateCollection(*models.Collection) error
	ListCollections() ([]*models.Collection, error)
	GetCollection(int) (*models.Collection, error)
	UpdateCollection(*models.Collection) error
	DeleteCollection(int) error

	// Curriculum
	SetCollection(*models.Collection) error

	// LiveLessons
	CreateLiveLesson(*models.LiveLesson) error
	ListLiveLessons() ([]*models.LiveLesson, error)
	GetLiveLesson(int) (*models.LiveLesson, error)
	UpdateLiveLesson(*models.LiveLesson) error
	DeleteLiveLesson(int) error
}

type AntidoteDB struct {
	User     string
	Password string
	Database string
}

// Check that the database exists, tables are in place, and that the version matches us
func (a *AntidoteDB) Preflight() error {
	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	var metaRaw []models.Meta
	_, err := db.Query(&metaRaw, `SELECT * FROM antidote_meta`)
	if err != nil {
		return err
	}

	meta := map[string]string{}
	for i := range metaRaw {
		meta[metaRaw[i].Key] = metaRaw[i].Key
	}

	return nil

}

func (a *AntidoteDB) Initialize() {

	// Connect to Postgres
	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	// TODO(mierdin): Acquire database lock here

	for _, model := range []interface{}{
		(*models.Meta)(nil),
		(*models.Lesson)(nil),
		(*models.LessonEndpoint)(nil),
	} {
		err := db.DropTable(model, &orm.DropTableOptions{
			// Temp: true,
		})
		if err != nil {
			panic(err)
		}
	}

	for _, model := range []interface{}{
		(*models.Meta)(nil),
		(*models.Lesson)(nil),
		(*models.LessonEndpoint)(nil),
	} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			// Temp: true,
		})
		if err != nil {
			panic(err)
		}
	}

}
