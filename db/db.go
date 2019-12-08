package db

import (
	"errors"
	"fmt"
	"strings"

	pg "github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	log "github.com/sirupsen/logrus"

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

	// NOTE
	// Delete and Update functions were intentionally not implemented for resource types, like lessons and collections.
	// The idea is that these should be coming from a git repo, so if you don't want them in the curriculum, don't have them in
	// the repo when you import.

	// Misc
	Preflight() error
	Initialize() error

	// Lessons
	ReadLessons() error
	InsertLesson([]*models.Lesson) error
	ListLessons() ([]*models.Lesson, error)
	GetLesson(string) (*models.Lesson, error)

	// // Collections
	// ReadCollections() error
	// InsertCollection([]*models.Collection) error
	// ListCollections() ([]*models.Collection, error)
	// GetCollection(string) (*models.Collection, error)
	// DeleteCollection(string) error

	// // Curriculum
	// SetCollection(*models.Collection) error

	// // LiveLessons
	// CreateLiveLesson(*models.LiveLesson) error
	// ListLiveLessons() ([]*models.LiveLesson, error)
	// GetLiveLesson(string) (*models.LiveLesson, error)
	// UpdateLiveLesson(*models.LiveLesson) error
	// DeleteLiveLesson(string) error

	// GCWhiteList

	// Sessions
}

// EnforceDBInterfaceCompliance forces AntidoteDB to conform to Databaser interface
// func EnforceDBInterfaceCompliance() {
// 	func(cr Databaser) {}(AntidoteDB{})
// }

type AntidoteDB struct {
	User            string
	Password        string
	Database        string
	AntidoteVersion string
	SyringeConfig   *config.SyringeConfig
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
	_, err := db.Query(&metaRaw, `SELECT * FROM meta`)
	if err != nil {
		return err
	}

	meta := map[string]string{}
	for i := range metaRaw {
		meta[metaRaw[i].Key] = metaRaw[i].Value
	}

	// Ensure the database was initialized with this version of Antidote
	if _, ok := meta["AntidoteVersion"]; ok {
		if meta["AntidoteVersion"] != a.AntidoteVersion {
			return errors.New(fmt.Sprintf("Database provisioned with different version of Antidote (expected %s, got %s). Re-run 'antidote import'", a.AntidoteVersion, meta["AntidoteVersion"]))
		}
	} else {
		return errors.New("Unable to retrieve version of database. Re-run 'antidote import'")
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
	} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			// Temp: true,
		})
		if err != nil {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	metaMap := map[string]string{
		"AntidoteVersion": a.AntidoteVersion,
	}
	for k, v := range metaMap {
		meta := &models.Meta{
			Key:   k,
			Value: v,
		}
		// log.Info("Inserting into meta: %v", meta)
		err := tx.Insert(meta)
		if err != nil {
			log.Errorf("Failed to insert meta information '%s' into the database: %v", meta.Key, err)
			return err
		}
	}
	tx.Commit()

	return nil

}
