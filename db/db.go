package db

import (
	"strings"

	pg "github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"

	config "github.com/nre-learning/syringe/config"
	models "github.com/nre-learning/syringe/db/models"
)

// TODO(mierdin): You probably just need a DB driver-esque interface that will be satisfied by
// go-pg. however, it is still useful to have a summary for dev time of implemented functions and those left
// to implement, which is kind of why you build this in the first place.

type DbDriver interface {
	Connect()
}

// Databaser defines all functions for the db layer
// Use this to provide a mock layer for tests
// TODO(mierdin): Enforce this somewhere
type Databaser interface {

	// Misc
	Preflight() error
	Initialize() error

	// Lessons
	ReadLessons() error
	InsertLesson([]*models.Lesson) error
	ListLessons() ([]*models.Lesson, error)
	// GetLesson(string) (*models.Lesson, error)
	// UpdateLesson(*models.Lesson) error  //TODO(mierdin): Probably not needed
	// DeleteLesson(string) error

	// // Collections
	// ReadCollections() error
	// InsertCollection([]*models.Collection) error
	// ListCollections() ([]*models.Collection, error)
	// GetCollection(string) (*models.Collection, error)
	// UpdateCollection(*models.Collection) error  //TODO(mierdin): Probably not needed
	// DeleteCollection(string) error

	// // Curriculum
	// SetCollection(*models.Collection) error

	// // LiveLessons
	// CreateLiveLesson(*models.LiveLesson) error
	// ListLiveLessons() ([]*models.LiveLesson, error)
	// GetLiveLesson(string) (*models.LiveLesson, error)
	// UpdateLiveLesson(*models.LiveLesson) error
	// DeleteLiveLesson(string) error
}

// EnforceDBInterfaceCompliance forces AntidoteDB to conform to Databaser interface
// func EnforceDBInterfaceCompliance() {
// 	func(cr Databaser) {}(AntidoteDB{})
// }

type AntidoteDB struct {
	User          string
	Password      string
	Database      string
	SyringeConfig *config.SyringeConfig
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

func (a *AntidoteDB) Initialize() error {

	// Connect to Postgres
	db := pg.Connect(&pg.Options{
		User:     a.User,
		Password: a.Password,
		Database: a.Database,
	})
	defer db.Close()

	// TODO(mierdin): Can we create database first, or at least check if it exists?

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
			if strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return err
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
			return err
		}
	}

	return nil

}
